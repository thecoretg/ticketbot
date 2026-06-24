package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/journal"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg         *models.Config
	CW          *cwsvc.Service
	Notifier    *notifier.Service
	Workflow    *workflow.Service
	Journal     *journal.Service
	ticketLocks sync.Map
}

func New(cfg *models.Config, cw *cwsvc.Service, ns *notifier.Service, wfs *workflow.Service, js *journal.Service) *Service {
	return &Service{
		Cfg:      cfg,
		CW:       cw,
		Notifier: ns,
		Workflow: wfs,
		Journal:  js,
	}
}

// ProcessTicket handles one CW ticket webhook. `added` reports whether the
// webhook action was "added" (a genuinely new ticket) vs "updated"; it is the
// authoritative new-vs-updated signal — local DB presence alone is not, because a
// ticket the bot never synced can still be an update (e.g. its "added" hook was
// missed). A ticket counts as new only when CW said "added" AND we haven't already
// synced it (so a re-delivered "added" hook doesn't re-fire new-ticket routing).
func (s *Service) ProcessTicket(ctx context.Context, id int, actorMemberID string, added bool) (err error) {
	start := time.Now()
	slog.Debug("ticketbot: request received", "ticket_id", id)

	// Accumulated for the ticket journal, recorded once in the defer (covers every
	// return path, including errors).
	var (
		isNew          bool
		full           *models.FullTicket
		events         []models.JournalEvent
		workflowRan    bool
		wfBotTriggered bool
	)

	defer func() {
		took := time.Since(start).Seconds()
		if err != nil {
			slog.Error("ticketbot: request finished with error", "ticket_id", id, "took_seconds", took, "error", err.Error())
		} else {
			slog.Debug("ticketbot: request finished", "ticket_id", id, "took_seconds", took)
		}
		botRun := botTriggeredRun(workflowRan, wfBotTriggered, full, s.Cfg.CwBotMemberIdentifier)
		s.recordJournal(ctx, id, isNew, full, events, err, start, botRun)
	}()

	// Prevent a ticket from processing multiple times to prevent duplicate notifications.
	// Connectwise frequently sends multiple hooks for the same ticket simultaneously.
	lock := s.getTicketLock(id)
	lock.Lock()
	defer lock.Unlock()

	exists, err := s.CW.Tickets.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking if ticket %d exists: %w", id, err)
	}
	isNew = added && !exists

	// Workflow pipeline: mutate the CW ticket before we sync it locally. No-op when
	// the flag is off, the service is unset, or no workflows match. Failures here are
	// deliberately non-fatal so a bad workflow never blocks sync/notify. A workflow
	// send_message action may request that the downstream notifier be skipped.
	var skipNotify bool
	if s.Cfg.AttemptWorkflow && s.Workflow != nil {
		workflowRan = true
		wfRes, werr := s.Workflow.Run(ctx, id, isNew, actorMemberID)
		if werr != nil {
			slog.Error("ticketbot: workflow pipeline failed", "ticket_id", id, "error", werr.Error())
			events = append(events, models.JournalEvent{Text: "Workflow error: " + werr.Error(), Status: models.JournalError})
		}
		if wfRes != nil {
			events = append(events, wfRes.Events...)
			skipNotify = wfRes.SkipNotify
			wfBotTriggered = wfRes.BotTriggered
		}
	}

	ticket, err := s.CW.ProcessTicket(ctx, id, "ticketbot")
	if err != nil {
		return fmt.Errorf("processing ticket %d: %w", id, err)
	}
	full = ticket

	// A workflow send_message action asked to suppress the normal notification so
	// it isn't doubled up. Record the skip so the note isn't re-notified later.
	if skipNotify {
		slog.Debug("ticketbot: notifier skipped by workflow", "ticket_id", id)
		events = append(events, models.JournalEvent{Text: "Skipped default notifications (workflow requested)", Status: models.JournalSkip})
		if serr := s.Notifier.AddSkippedNotification(ctx, ticket, "workflow send_message"); serr != nil {
			slog.Error("ticketbot: recording workflow-skipped notification", "ticket_id", id, "error", serr.Error())
		}
		return nil
	}

	if s.Cfg.AttemptNotify {
		nRes, nerr := s.Notifier.Run(ctx, ticket, isNew)
		if nRes != nil {
			events = append(events, nRes.Events...)
		}
		if nerr != nil {
			return fmt.Errorf("running notifier for ticket %d: %w", id, nerr)
		}
		return nil
	}

	slog.Debug("ticketbot: attempt notify disabled", "ticket_id", id)
	events = append(events, models.JournalEvent{Text: "Notifications disabled", Status: models.JournalSkip})
	if serr := s.Notifier.AddSkippedNotification(ctx, ticket, "ticketbot"); serr != nil {
		return fmt.Errorf("skipping notification for ticket %d: %w", id, serr)
	}

	return nil
}

// recordJournal writes one timeline run for the ticket, unless the run was
// triggered by the bot's own prior edit (a loop echo), which we don't audit.
func (s *Service) recordJournal(ctx context.Context, id int, isNew bool, full *models.FullTicket, events []models.JournalEvent, err error, start time.Time, botRun bool) {
	if s.Journal == nil || botRun {
		return
	}

	run := buildRun(start, isNew, events, err)
	if rerr := s.Journal.Record(ctx, id, full, run); rerr != nil {
		slog.Error("ticketbot: recording ticket journal", "ticket_id", id, "error", rerr.Error())
	}
}

// botTriggeredRun reports whether this webhook was triggered by the bot's own
// prior edit (a loop echo we don't journal). When workflows ran, their pre-action
// signal is authoritative — the post-sync ticket would falsely look bot-edited
// because this run's own ticket_update/add_note actions set the bot as editor.
// When workflows didn't run, nothing in this run touched the ticket, so the synced
// editor is accurate.
func botTriggeredRun(workflowRan, wfBotTriggered bool, full *models.FullTicket, botMember string) bool {
	if workflowRan {
		return wfBotTriggered
	}
	return full != nil && botMember != "" && full.Ticket.UpdatedBy != nil && *full.Ticket.UpdatedBy == botMember
}

// buildRun assembles a TicketRun from the accumulated events and any fatal error.
func buildRun(start time.Time, isNew bool, events []models.JournalEvent, err error) models.TicketRun {
	trigger := models.TriggerUpdated
	if isNew {
		trigger = models.TriggerNew
	}

	if err != nil && len(events) == 0 {
		events = append(events, models.JournalEvent{Text: "Processing failed: " + err.Error(), Status: models.JournalError})
	}

	hadError := err != nil
	hasOK := false
	for _, e := range events {
		switch e.Status {
		case models.JournalError:
			hadError = true
		case models.JournalOK:
			hasOK = true
		}
	}

	outcome := models.OutcomeNothingToDo
	switch {
	case hadError:
		outcome = models.OutcomeWithErrors
	case hasOK:
		outcome = models.OutcomeCompleted
	}

	return models.TicketRun{
		Time:     start,
		Trigger:  trigger,
		Events:   events,
		Outcome:  outcome,
		HadError: hadError,
	}
}

func (s *Service) getTicketLock(id int) *sync.Mutex {
	li, _ := s.ticketLocks.LoadOrStore(id, &sync.Mutex{})
	return li.(*sync.Mutex)
}
