package ticketbot

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

// run collects the events for one intake pass and flushes them in a single batch. When a workflow
// ran it also carries what the results page summarises.
type run struct {
	id       string
	ticketID int
	source   models.EventSource
	dryRun   bool
	events   []*models.TicketEvent
	now      func() time.Time
	started  time.Time

	// Set by runWorkflow and sendNotifications when an enabled workflow was found.
	wf      *models.Workflow
	res     *workflow.Result
	notifs  []notifier.Outcome
	boardID int
	// errors counts error events added to the run that are not action or notification outcomes.
	errors int
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
		started:  now(),
	}
}

// summary is the results-page row for this run, or nil when no enabled workflow ran.
func (r *run) summary() *models.WorkflowRun {
	if r.wf == nil || r.res == nil {
		return nil
	}
	id := r.wf.ID
	s := &models.WorkflowRun{
		RunID:        r.id,
		TicketID:     r.ticketID,
		BoardID:      r.boardID,
		WorkflowID:   &id,
		WorkflowName: r.wf.Name,
		Event:        r.res.Event,
		Source:       r.source,
		DryRun:       r.dryRun,
		StartedAt:    r.started,
		DurationMs:   int(r.now().Sub(r.started) / time.Millisecond),
		Steps:        len(r.res.Steps),
		Actions:      len(r.res.Actions),
		Writes:       r.res.CWWrites,
		Errors:       r.errors,
	}
	for _, a := range r.res.Actions {
		if a.Result == workflow.ResultError {
			s.Errors++
		}
	}
	for _, n := range r.notifs {
		switch n.Result {
		case notifier.ResultSent:
			s.NotifSent++
		case notifier.ResultWouldSend:
			s.NotifWouldSend++
		case notifier.ResultNoRecipients:
			s.NotifNone++
		case notifier.ResultError:
			s.Errors++
		}
	}
	s.Outcome = s.Verdict()
	return s
}

func (r *run) add(kind models.EventKind, payload any) {
	if kind == models.EventError {
		r.errors++
	}
	e, err := models.NewTicketEvent(r.ticketID, r.id, kind, r.source, r.dryRun, payload, r.now())
	if err != nil {
		slog.Error("ticketbot: marshalling event payload", "ticket_id", r.ticketID, "kind", kind, "error", err.Error())
		return
	}

	r.events = append(r.events, e)
}

// flush writes the run's events and, when a workflow ran, its summary row. The summary is only
// worth writing when the events landed, because it points at them.
func (r *run) flush(ctx context.Context, repo repos.TicketEventRepository, runs repos.WorkflowRunRepository) {
	if len(r.events) == 0 {
		return
	}

	if err := repo.InsertBatch(ctx, r.events); err != nil {
		slog.Error("ticketbot: inserting ticket events", "ticket_id", r.ticketID, "run_id", r.id, "count", len(r.events), "error", err.Error())
		return
	}
	if s := r.summary(); s != nil && runs != nil {
		if err := runs.Insert(ctx, s); err != nil {
			slog.Error("ticketbot: inserting workflow run summary", "ticket_id", r.ticketID, "run_id", r.id, "error", err.Error())
		}
	}
}

func newRunID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().UTC().Format("20060102150405.000000000")
	}

	return hex.EncodeToString(b[:])
}
