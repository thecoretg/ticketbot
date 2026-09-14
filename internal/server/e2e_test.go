package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/models"
)

// TestE2EProcessTicket runs a real intake against ConnectWise (read-only) and the database in
// POSTGRES_DSN. It needs the usual server env plus TEST_TICKET_IDS, and is skipped otherwise.
// The second pass over the same ticket must be a silent no-op.
func TestE2EProcessTicket(t *testing.T) {
	ids := os.Getenv("TEST_TICKET_IDS")
	if ids == "" || os.Getenv("POSTGRES_DSN") == "" {
		t.Skip("TEST_TICKET_IDS / POSTGRES_DSN not set")
	}
	id, err := strconv.Atoi(strings.Split(ids, ",")[0])
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	level := new(slog.LevelVar)
	logBuf := logging.NewBufferHandler(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}), 100)
	app, _, err := NewApp(ctx, 8, level, logBuf)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Pool.Close()

	// start clean so the first pass is a "created"
	if err := app.Svc.CW.PermDeleteTicket(ctx, id); err != nil {
		t.Fatal(err)
	}

	opts := ticketbot.ProcessOpts{Source: models.SourceManual, RunRules: false}
	if err := app.Svc.Ticketbot.ProcessTicket(ctx, id, opts); err != nil {
		t.Fatal(err)
	}

	events, err := app.Stores.TicketEvents.ListByTicket(ctx, id, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].Kind != models.EventCreated {
		t.Fatalf("after first pass want 1 created event, got %+v", events)
	}
	var p models.ChangePayload
	if err := json.Unmarshal(events[0].Payload, &p); err != nil {
		t.Fatal(err)
	}
	t.Logf("created payload: %s", events[0].Payload)

	stored, err := app.Stores.CW.Ticket.Get(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Raw) == 0 {
		t.Error("raw JSON not stored")
	}
	if p.NewNote != nil && (stored.LatestNoteID == nil || *stored.LatestNoteID != p.NewNote.ID) {
		t.Errorf("latest_note_id %v does not match new_note %d", stored.LatestNoteID, p.NewNote.ID)
	}

	// second pass: nothing changed, no new events
	if err := app.Svc.Ticketbot.ProcessTicket(ctx, id, opts); err != nil {
		t.Fatal(err)
	}
	events, err = app.Stores.TicketEvents.ListByTicket(ctx, id, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("second pass should add no events, got %d", len(events))
	}

	detail, err := app.Svc.CW.GetTicketDetail(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Ticket.BoardName == "" || detail.Ticket.CWURL == "" || len(detail.Events) != 1 {
		t.Errorf("detail incomplete: %+v", detail.Ticket)
	}

	// simulate a stored change so the third pass records an update
	stored.Summary = stored.Summary + " (stale)"
	stored.Raw = nil
	if _, err := app.Stores.CW.Ticket.Upsert(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if err := app.Svc.Ticketbot.ProcessTicket(ctx, id, opts); err != nil {
		t.Fatal(err)
	}
	events, err = app.Stores.TicketEvents.ListByTicket(ctx, id, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 || events[0].Kind != models.EventUpdated {
		t.Fatalf("third pass want updated event, got %+v", events)
	}
	t.Logf("updated payload: %s", events[0].Payload)
	var up models.ChangePayload
	if err := json.Unmarshal(events[0].Payload, &up); err != nil {
		t.Fatal(err)
	}
	if len(up.Changes) != 1 || up.Changes[0].Field != "summary" {
		t.Errorf("summary change not recorded: %s", events[0].Payload)
	}
}
