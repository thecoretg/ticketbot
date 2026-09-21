package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/thecoretg/ticketbot/internal/env"
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
	e, err := env.Load()
	if err != nil {
		t.Fatal(err)
	}
	app, _, err := NewApp(ctx, e, 11, level, logBuf)
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

	// fourth pass: run a dry-run workflow with notify + add_note against a forced change.
	// Master dry run guarantees nothing is written to ConnectWise or Webex.
	dry := true
	if _, err := app.Svc.Config.Update(ctx, &models.ConfigUpdateParams{MasterDryRun: &dry}); err != nil {
		t.Fatal(err)
	}
	room, err := app.Stores.WebexRecipients.Upsert(ctx, &models.WebexRecipient{WebexID: "e2e-room", Name: "E2E Room", Type: models.RecipientTypeRoom})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stores.WebexRecipients.Delete(ctx, room.ID) })

	if existing, err := app.Stores.Workflows.GetByBoard(ctx, stored.BoardID); err == nil {
		_ = app.Stores.Workflows.Delete(ctx, existing.ID)
	}
	wf, err := app.Svc.Workflow.Create(ctx, &models.Workflow{BoardID: stored.BoardID, Enabled: true,
		Nodes: []models.Node{
			{ID: "t", Kind: models.NodeTrigger, Title: "updated", Enabled: true, Events: []models.TriggerEvent{models.TriggerUpdated}},
			{ID: "c", Kind: models.NodeIf, Title: "e2e", Enabled: true, Condition: "changed/summary = true"},
			{ID: "n1", Kind: models.NodeKind(models.ActionNotify), Title: "room", Enabled: true, ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Channel: models.ChannelWebexRoom, RecipientID: &room.ID}}},
			{ID: "a", Kind: models.NodeKind(models.ActionAddNote), Title: "note", Enabled: true, ActionSettings: models.ActionSettings{AddNote: &models.AddNoteAction{Text: "e2e dry run", Internal: true}}},
			{ID: "n2", Kind: models.NodeKind(models.ActionNotify), Title: "owner", Enabled: true, ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Channel: models.ChannelResourcesOwner}}},
		},
		Edges: []models.Edge{
			{From: "t", To: "c", Port: models.PortOut},
			{From: "c", To: "n1", Port: models.PortYes},
			{From: "n1", To: "a", Port: models.PortOut},
			{From: "a", To: "n2", Port: models.PortOut},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = app.Stores.Workflows.Delete(ctx, wf.ID) })

	stored, _ = app.Stores.CW.Ticket.Get(ctx, id)
	stored.Summary += " (stale again)"
	stored.Raw = nil
	if _, err := app.Stores.CW.Ticket.Upsert(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if err := app.Svc.Ticketbot.ProcessTicket(ctx, id, ticketbot.ProcessOpts{Source: models.SourceManual, RunRules: true}); err != nil {
		t.Fatal(err)
	}

	events, err = app.Stores.TicketEvents.ListByTicket(ctx, id, 20, nil)
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[models.EventKind]int{}
	for _, e := range events {
		if e.RunID == events[0].RunID {
			kinds[e.Kind]++
			if e.Kind != models.EventUpdated && !e.DryRun {
				t.Errorf("event %s should be flagged dry_run", e.Kind)
			}
			t.Logf("%s: %s", e.Kind, e.Payload)
		}
	}
	if kinds[models.EventUpdated] != 1 || kinds[models.EventWorkflow] != 1 || kinds[models.EventAction] != 3 || kinds[models.EventNotification] < 1 {
		t.Errorf("unexpected event mix for dry run: %v", kinds)
	}
	for _, e := range events {
		if e.RunID != events[0].RunID || e.Kind != models.EventAction {
			continue
		}
		var ap models.ActionPayload
		_ = json.Unmarshal(e.Payload, &ap)
		if ap.Kind == string(models.ActionAddNote) && ap.Result != "would_run" {
			t.Errorf("add_note in dry run should be would_run: %+v", ap)
		}
	}
	notifs, err := app.Stores.TicketNotifications.ListAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range notifs {
		if n.TicketID == id {
			t.Errorf("dry run must not store ticket_notification rows: %+v", n)
		}
	}
}
