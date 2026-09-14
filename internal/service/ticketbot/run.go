package ticketbot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// run collects the events for one intake pass and flushes them in a single batch.
type run struct {
	id       string
	ticketID int
	source   models.EventSource
	dryRun   bool
	events   []*models.TicketEvent
	now      func() time.Time
}

func newRun(ticketID int, source models.EventSource, now func() time.Time) *run {
	if now == nil {
		now = time.Now
	}

	return &run{
		id:       newRunID(),
		ticketID: ticketID,
		source:   source,
		now:      now,
	}
}

func (r *run) add(kind models.EventKind, payload any) {
	e, err := models.NewTicketEvent(r.ticketID, r.id, kind, r.source, r.dryRun, payload, r.now())
	if err != nil {
		slog.Error("ticketbot: marshalling event payload", "ticket_id", r.ticketID, "kind", kind, "error", err.Error())
		return
	}

	r.events = append(r.events, e)
}

func (r *run) flush(ctx context.Context, repo repos.TicketEventRepository) {
	if len(r.events) == 0 {
		return
	}

	if err := repo.InsertBatch(ctx, r.events); err != nil {
		slog.Error("ticketbot: inserting ticket events", "ticket_id", r.ticketID, "run_id", r.id, "count", len(r.events), "error", err.Error())
	}
}

func newRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(b[:])
}
