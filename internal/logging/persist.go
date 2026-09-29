package logging

import (
	"context"
	"log/slog"
	"time"
)

// LogPersistRepository is the subset of the log repo the persister needs.
type LogPersistRepository interface {
	InsertBatch(ctx context.Context, entries []LogEntry) error
	GetRecent(ctx context.Context, limit int) ([]LogEntry, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}

// LogConfig is the subset of app config the persister reads.
type LogConfig interface {
	GetLogRetentionDays() int
}

// Persister batches log entries from a BufferHandler into the DB. Its Purge enforces the
// retention policy on the intake service's hourly purge tick.
type Persister struct {
	repo LogPersistRepository
	buf  *BufferHandler
	cfg  LogConfig
	ch   chan LogEntry
}

const (
	flushInterval = 5 * time.Second
	channelBuf    = 2000
)

// NewPersister wires the persister to the buffer handler.
// Call Start to launch the background goroutines.
func NewPersister(repo LogPersistRepository, buf *BufferHandler, cfg LogConfig) *Persister {
	ch := make(chan LogEntry, channelBuf)
	buf.persistCh = ch
	return &Persister{repo: repo, buf: buf, cfg: cfg, ch: ch}
}

// SeedBuffer reads the most recent entries from the DB into the ring buffer on startup.
func (p *Persister) SeedBuffer(ctx context.Context) error {
	entries, err := p.repo.GetRecent(ctx, p.buf.r.size)
	if err != nil {
		return err
	}
	p.buf.Seed(entries)
	return nil
}

// Start launches the writer goroutine. It returns immediately; the writer stops when ctx is
// cancelled.
func (p *Persister) Start(ctx context.Context) {
	go p.runWriter(ctx)
}

func (p *Persister) runWriter(ctx context.Context) {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	var batch []LogEntry
	flush := func() {
		if len(batch) == 0 {
			return
		}
		if err := p.repo.InsertBatch(ctx, batch); err != nil {
			slog.Warn("log persister: failed to write batch", "error", err, "count", len(batch))
		}
		batch = batch[:0]
	}

	for {
		select {
		case e := <-p.ch:
			batch = append(batch, e)
		case <-ticker.C:
			flush()
		case <-ctx.Done():
			// drain remaining entries before exiting
			for {
				select {
				case e := <-p.ch:
					batch = append(batch, e)
				default:
					flush()
					return
				}
			}
		}
	}
}

// Purge implements intake.Purger: log rows older than the retention setting go. 0 keeps them.
func (p *Persister) Purge(ctx context.Context, now time.Time) {
	retentionDays := p.cfg.GetLogRetentionDays()
	if retentionDays <= 0 {
		return
	}
	n, err := p.repo.DeleteOlderThan(ctx, now.AddDate(0, 0, -retentionDays))
	if err != nil {
		slog.Warn("log persister: cleanup failed", "error", err)
		return
	}
	if n > 0 {
		slog.Info("log persister: cleaned up old entries", "deleted", n, "retention_days", retentionDays)
	}
}
