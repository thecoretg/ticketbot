package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Run executes the enabled workflows against a ticket before it is synced
// locally. It is a no-op when no workflows match. Per-action failures are logged
// and skipped so a single bad action never aborts the pipeline; a returned error
// signals an infrastructure failure (DB / initial ticket fetch) that the caller
// logs but does not treat as fatal.
//
// The returned RunResult reports whether the downstream notifier should be
// suppressed and carries human-readable events for the ticket journal. A bot-
// authored webhook returns an empty result (the pipeline is skipped).
func (s *Service) Run(ctx context.Context, ticketID int, isNew bool, actorMemberID string) (*RunResult, error) {
	res := &RunResult{}

	workflows, err := s.Workflows.ListEnabled(ctx)
	if err != nil {
		return res, fmt.Errorf("listing enabled workflows: %w", err)
	}
	if len(workflows) == 0 {
		return res, nil
	}

	t, err := s.CWClient.GetTicket(ctx, ticketID, nil)
	if err != nil {
		return res, fmt.Errorf("getting ticket %d: %w", ticketID, err)
	}

	// The most recent note feeds last-note conditions and the send_message ticket
	// card. A ticket with no notes is not an error.
	note, err := s.CWClient.GetMostRecentTicketNote(ctx, ticketID)
	if err != nil && !errors.Is(err, psa.ErrNotFound) {
		slog.Warn("workflow: fetching most recent note", "ticket_id", ticketID, "error", err.Error())
		note = nil
	}

	slog.Debug("workflow: pipeline invoked", "ticket_id", ticketID, "is_new", isNew, "webhook_member", actorMemberID, "updated_by", t.Info.UpdatedBy)

	// Loop prevention: if the ticket's most recent editor is the bot itself, this
	// webhook was triggered by our own change — skip the pipeline. Connectwise
	// callback payloads always report the callback-owner member rather than the
	// actual editor, so we key off the ticket's updatedBy field (which CW sets to
	// our API member when the bot's patch lands) instead of the webhook MemberID.
	if s.Cfg.CwBotMemberIdentifier != "" && t.Info.UpdatedBy == s.Cfg.CwBotMemberIdentifier {
		slog.Debug("workflow: skipping bot-authored update", "ticket_id", ticketID, "updated_by", t.Info.UpdatedBy)
		res.BotTriggered = true
		return res, nil
	}

	ctxData := EvalCtx{Ticket: t, LastNote: note}
	ex := s.exec(note)

	for _, wf := range workflows {
		if !workflowApplies(wf, ctxData, isNew) {
			continue
		}
		res.Events = append(res.Events, matchedEvent(wf))
		s.runActions(ctx, wf, ex, t, ticketID, res)
	}

	return res, nil
}

// runActions runs a matched workflow's actions in order, appending a timeline
// event for each and setting SkipNotify on the result.
func (s *Service) runActions(ctx context.Context, wf *models.Workflow, ex *Exec, t *psa.Ticket, ticketID int, res *RunResult) {
	for i, a := range wf.Actions {
		handler, ok := s.registry[a.Type]
		if !ok {
			slog.Warn("workflow: unknown action", "workflow_id", wf.ID, "action_index", i, "type", a.Type)
			continue
		}

		// Run-once markers for non-idempotent actions (notes, messages).
		if !handler.Idempotent() {
			ran, err := s.Runs.Exists(ctx, ticketID, wf.ID, i)
			if err != nil {
				slog.Error("workflow: checking run marker", "workflow_id", wf.ID, "action_index", i, "ticket_id", ticketID, "error", err.Error())
				continue
			}
			if ran {
				continue
			}
		}

		params := handler.NewParams()
		if err := json.Unmarshal(rawConfig(a.Config), params); err != nil {
			slog.Error("workflow: bad action config", "workflow_id", wf.ID, "action_index", i, "type", a.Type, "error", err.Error())
			res.Events = append(res.Events, errEvent("Action could not run (%s): bad configuration", actionLabel(a.Type)))
			continue
		}
		if err := renderTemplatesFor(params, t); err != nil {
			slog.Error("workflow: template render failed", "workflow_id", wf.ID, "action_index", i, "type", a.Type, "error", err.Error())
			res.Events = append(res.Events, errEvent("Action could not run (%s): template error", actionLabel(a.Type)))
			continue
		}

		change, err := handler.Apply(ctx, ex, t, params)
		if err != nil {
			slog.Error("workflow: apply failed", "workflow_id", wf.ID, "action_index", i, "type", a.Type, "ticket_id", ticketID, "error", err.Error())
			res.Events = append(res.Events, errEvent("%s failed: %s", actionLabel(a.Type), err.Error()))
			continue
		}

		if !handler.Idempotent() {
			if err := s.Runs.Insert(ctx, ticketID, wf.ID, i); err != nil {
				slog.Error("workflow: recording run marker", "workflow_id", wf.ID, "action_index", i, "ticket_id", ticketID, "error", err.Error())
			}
		}

		if change.SkipNotify {
			res.SkipNotify = true
		}
		res.Events = append(res.Events, actionEvent(a.Type, change))

		slog.Debug("workflow: action processed", "workflow_id", wf.ID, "action_index", i, "type", a.Type, "ticket_id", ticketID, "changed", change.Applied)
	}
}

// workflowApplies reports whether a workflow's board, on-ticket-action trigger,
// and condition tree all match the ticket.
func workflowApplies(wf *models.Workflow, c EvalCtx, isNew bool) bool {
	if wf.CwBoardID != c.Ticket.Board.ID {
		return false
	}

	switch wf.OnTicketAction {
	case models.WorkflowOnNew:
		if !isNew {
			return false
		}
	case models.WorkflowOnUpdated:
		if isNew {
			return false
		}
	}

	return evalGroup(wf.Root, c)
}

func rawConfig(c json.RawMessage) []byte {
	if len(c) == 0 {
		return []byte("{}")
	}
	return c
}
