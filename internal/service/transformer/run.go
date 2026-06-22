package transformer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Run executes the enabled transformer rules against a ticket before it is
// synced locally. It is a no-op when no rules match. Per-rule failures are
// logged and skipped so a single bad rule never aborts the pipeline; a returned
// error signals an infrastructure failure (DB / initial ticket fetch) that the
// caller logs but does not treat as fatal.
func (s *Service) Run(ctx context.Context, ticketID int, isNew bool, actorMemberID string) error {
	rules, err := s.Rules.ListEnabled(ctx)
	if err != nil {
		return fmt.Errorf("listing enabled transformer rules: %w", err)
	}
	if len(rules) == 0 {
		return nil
	}

	t, err := s.CWClient.GetTicket(ctx, ticketID, nil)
	if err != nil {
		return fmt.Errorf("getting ticket %d: %w", ticketID, err)
	}

	slog.Debug("transformer: pipeline invoked", "ticket_id", ticketID, "is_new", isNew, "webhook_member", actorMemberID, "updated_by", t.Info.UpdatedBy)

	// Layer A loop prevention: if the ticket's most recent editor is the bot
	// itself, this webhook was triggered by our own change — skip the pipeline.
	// Connectwise callback payloads always report the callback-owner member rather
	// than the actual editor, so we key off the ticket's updatedBy field (which CW
	// sets to our API member when the bot's patch lands) instead of the webhook MemberID.
	if s.Cfg.CwBotMemberIdentifier != "" && t.Info.UpdatedBy == s.Cfg.CwBotMemberIdentifier {
		slog.Debug("transformer: skipping bot-authored update", "ticket_id", ticketID, "updated_by", t.Info.UpdatedBy)
		return nil
	}

	ex := s.exec()
	for _, r := range rules {
		if !ruleApplies(r, t, isNew) {
			continue
		}

		tf, ok := s.registry[r.Action]
		if !ok {
			slog.Warn("transformer: unknown action", "rule_id", r.ID, "action", r.Action)
			continue
		}

		// Layer D: run-once markers for non-idempotent actions (e.g. notes).
		if !tf.Idempotent() {
			ran, err := s.Runs.Exists(ctx, ticketID, r.ID)
			if err != nil {
				slog.Error("transformer: checking run marker", "rule_id", r.ID, "ticket_id", ticketID, "error", err.Error())
				continue
			}
			if ran {
				continue
			}
		}

		params := tf.NewParams()
		if err := json.Unmarshal(rawConfig(r.Config), params); err != nil {
			slog.Error("transformer: bad rule config", "rule_id", r.ID, "action", r.Action, "error", err.Error())
			continue
		}
		if err := renderParams(params, t); err != nil {
			slog.Error("transformer: template render failed", "rule_id", r.ID, "action", r.Action, "error", err.Error())
			continue
		}

		change, err := tf.Apply(ctx, ex, t, params)
		if err != nil {
			slog.Error("transformer: apply failed", "rule_id", r.ID, "action", r.Action, "ticket_id", ticketID, "error", err.Error())
			continue
		}

		if !tf.Idempotent() {
			if err := s.Runs.Insert(ctx, ticketID, r.ID); err != nil {
				slog.Error("transformer: recording run marker", "rule_id", r.ID, "ticket_id", ticketID, "error", err.Error())
			}
		}

		slog.Debug("transformer: rule processed", "rule_id", r.ID, "action", r.Action, "ticket_id", ticketID, "changed", change.Applied)
	}

	return nil
}

// ruleApplies reports whether a rule's board, apply-on, and field conditions all
// match the ticket.
func ruleApplies(r *models.TransformerRule, t *psa.Ticket, isNew bool) bool {
	if r.CwBoardID != nil && *r.CwBoardID != t.Board.ID {
		return false
	}

	switch r.ApplyOn {
	case models.TransformerApplyNew:
		if !isNew {
			return false
		}
	case models.TransformerApplyUpdated:
		if isNew {
			return false
		}
	}

	// All conditions must match (AND).
	for _, c := range r.Conditions {
		if !evalCondition(c, t) {
			return false
		}
	}

	return true
}

func rawConfig(c json.RawMessage) []byte {
	if len(c) == 0 {
		return []byte("{}")
	}
	return c
}
