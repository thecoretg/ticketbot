package logging

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"
)

const defaultBufferSize = 500

// LogEntry is a single buffered log record.
type LogEntry struct {
	Time    time.Time      `json:"time"`
	Level   string         `json:"level"`
	Message string         `json:"message"`
	Attrs   map[string]any `json:"attrs,omitempty"`
}

// ring holds the shared mutable state so all derived handlers write to the same buffer.
type ring struct {
	mu    sync.Mutex
	buf   []LogEntry
	size  int
	head  int
	count int
}

func (r *ring) push(e LogEntry) {
	r.mu.Lock()
	r.buf[r.head] = e
	r.head = (r.head + 1) % r.size
	if r.count < r.size {
		r.count++
	}
	r.mu.Unlock()
}

func (r *ring) entries() []LogEntry {
	r.mu.Lock()
	defer r.mu.Unlock()

	out := make([]LogEntry, r.count)
	if r.count < r.size {
		copy(out, r.buf[:r.count])
	} else {
		n := copy(out, r.buf[r.head:])
		copy(out[n:], r.buf[:r.head])
	}
	return out
}

func (r *ring) resize(newSize int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// collect existing entries in order (under lock, without re-locking)
	old := make([]LogEntry, r.count)
	if r.count < r.size {
		copy(old, r.buf[:r.count])
	} else {
		n := copy(old, r.buf[r.head:])
		copy(old[n:], r.buf[:r.head])
	}

	// trim to newSize if shrinking
	if len(old) > newSize {
		old = old[len(old)-newSize:]
	}

	// replace internals in-place so all derived handlers see the change
	r.buf = make([]LogEntry, newSize)
	copy(r.buf, old)
	r.size = newSize
	r.count = len(old)
	r.head = r.count % newSize
}

// BufferHandler wraps an existing slog.Handler and stores the last N entries.
type BufferHandler struct {
	inner      slog.Handler
	r          *ring
	preAttrs   []slog.Attr // attrs accumulated via WithAttrs
	groupStack []string    // active group names via WithGroup
	persistCh  chan<- LogEntry
}

// NewBufferHandler wraps inner and retains up to size log entries.
func NewBufferHandler(inner slog.Handler, size int) *BufferHandler {
	if size <= 0 {
		size = defaultBufferSize
	}
	return &BufferHandler{
		inner: inner,
		r:     &ring{buf: make([]LogEntry, size), size: size},
	}
}

func (h *BufferHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

func (h *BufferHandler) Handle(ctx context.Context, rec slog.Record) error {
	// always forward to the underlying handler (stdout/cloudwatch)
	if err := h.inner.Handle(ctx, rec); err != nil {
		return err
	}

	// exclude healthcheck and log-poll noise from the ring buffer and DB
	if strings.Contains(rec.Message, "/healthcheck") || strings.Contains(rec.Message, "/logs") {
		return nil
	}

	attrs := make(map[string]any)

	// merge pre-attached attrs (from WithAttrs / WithGroup calls)
	for _, a := range h.preAttrs {
		addAttrToMap(attrs, a)
	}

	// merge call-site attrs
	rec.Attrs(func(a slog.Attr) bool {
		addAttrToMap(attrs, a)
		return true
	})

	entry := LogEntry{
		Time:    rec.Time,
		Level:   rec.Level.String(),
		Message: rec.Message,
	}
	if len(attrs) > 0 {
		entry.Attrs = attrs
	}

	h.r.push(entry)
	if h.persistCh != nil {
		select {
		case h.persistCh <- entry:
		default: // drop if channel is full rather than block
		}
	}
	return nil
}

func (h *BufferHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, len(h.preAttrs)+len(attrs))
	copy(merged, h.preAttrs)

	// if we're inside a group, wrap the new attrs under the current group
	if len(h.groupStack) > 0 {
		group := attrs
		for i := len(h.groupStack) - 1; i >= 0; i-- {
			group = []slog.Attr{slog.Group(h.groupStack[i], attrsToAny(group)...)}
		}
		copy(merged[len(h.preAttrs):], group)
	} else {
		copy(merged[len(h.preAttrs):], attrs)
	}

	return &BufferHandler{
		inner:     h.inner.WithAttrs(attrs),
		r:         h.r,
		preAttrs:  merged,
		persistCh: h.persistCh,
	}
}

func (h *BufferHandler) WithGroup(name string) slog.Handler {
	stack := make([]string, len(h.groupStack)+1)
	copy(stack, h.groupStack)
	stack[len(h.groupStack)] = name

	return &BufferHandler{
		inner:      h.inner.WithGroup(name),
		r:          h.r,
		preAttrs:   h.preAttrs,
		groupStack: stack,
		persistCh:  h.persistCh,
	}
}

// Seed loads entries into the ring buffer without sending them to persistCh.
// Used to restore state from the DB on startup.
func (h *BufferHandler) Seed(entries []LogEntry) {
	for _, e := range entries {
		h.r.push(e)
	}
}

// Resize changes the ring buffer capacity in-place so all derived handlers
// (from WithAttrs/WithGroup) automatically see the new size.
// Existing entries are preserved; oldest are dropped if newSize is smaller.
func (h *BufferHandler) Resize(newSize int) {
	if newSize <= 0 || newSize == h.r.size {
		return
	}
	h.r.resize(newSize)
}

// Size returns the current buffer capacity.
func (h *BufferHandler) Size() int { return h.r.size }

// Entries returns buffered entries in chronological order (oldest first).
func (h *BufferHandler) Entries() []LogEntry {
	return h.r.entries()
}

func attrsToAny(attrs []slog.Attr) []any {
	out := make([]any, len(attrs))
	for i, a := range attrs {
		out[i] = a
	}
	return out
}
