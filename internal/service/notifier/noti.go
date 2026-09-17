package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

// Send results.
const (
	ResultSent      = "sent"
	ResultWouldSend = "would_send"
	ResultError     = "error"
)

// SendRequest is a batch of notify intents produced by one workflow run.
type SendRequest struct {
	Ticket  *models.FullTicket
	IsNew   bool
	DryRun  bool
	Intents []workflow.NotifyIntent
}

// Outcome is what happened for one recipient.
type Outcome struct {
	Rule          workflow.RuleRef
	Recipient     *models.WebexRecipient
	ForwardedFrom []string
	Result        string
	Err           error
	Notification  *models.TicketNotification
}

// Send resolves every intent to recipients, applies forwards once over the union, and delivers one
// message per recipient. In dry run nothing is sent and nothing is stored. Per-recipient failures
// are reported in the outcomes; the returned error covers only failures that stop the whole batch.
func (s *Service) Send(ctx context.Context, req SendRequest) ([]Outcome, error) {
	if req.Ticket == nil {
		return nil, errors.New("nil ticket received")
	}
	if len(req.Intents) == 0 {
		return nil, nil
	}

	t := req.Ticket
	logger := slog.Default().With("ticket_id", t.Ticket.ID, "dry_run", req.DryRun)

	// union natural recipients across intents; the first rule to name a recipient gets credit
	// (and its message template)
	natural := make(recipMap)
	attribution := make(map[int]workflow.NotifyIntent)
	var outcomes []Outcome

	for _, in := range req.Intents {
		recips, err := s.resolveTarget(ctx, t, in.Target)
		if err != nil {
			logger.Error("notifier: resolving notify target", "rule", in.Rule.RuleName, "target", in.Target.Target, "error", err.Error())
			outcomes = append(outcomes, Outcome{Rule: in.Rule, Result: ResultError, Err: err})
			continue
		}
		for id, r := range recips {
			if _, seen := natural[id]; seen {
				continue
			}
			natural[id] = r
			attribution[id] = in
		}
	}

	if len(natural) == 0 {
		logger.Debug("notifier: no recipients to send to")
		return outcomes, nil
	}

	final := s.applyForwards(ctx, t, natural)
	msgs := s.makeTicketMessages(t, final.toSlice(), req.IsNew, attribution)

	for _, m := range msgs {
		out := Outcome{
			Rule:      attribution[m.WebexRecipient.origin().ID].Rule,
			Recipient: m.WebexRecipient.recipient,
		}
		for _, f := range m.WebexRecipient.forwardChain {
			out.ForwardedFrom = append(out.ForwardedFrom, f.Name)
		}

		if req.DryRun {
			out.Result = ResultWouldSend
			outcomes = append(outcomes, out)
			continue
		}

		sent := s.sendNotification(ctx, &m)
		out.Notification = sent.Notification
		if sent.SendError != nil {
			out.Result, out.Err = ResultError, sent.SendError
		} else {
			out.Result = ResultSent
		}
		outcomes = append(outcomes, out)
	}

	logger.Info("notifier: notifications processed", "recipients", len(msgs))
	return outcomes, nil
}

func (s *Service) sendNotification(ctx context.Context, m *Message) *Message {
	logger := slog.Default().With(
		slog.Int("ticket_id", m.Notification.TicketID),
		slog.String("recipient", m.WebexRecipient.recipient.Name),
	)

	logger.Debug("notifier: sending notification")
	if _, err := s.MessageSender.PostMessage(ctx, &m.WebexMsg); err != nil {
		m.SendError = fmt.Errorf("sending webex message: %w", err)
	}

	// record the attempt even on failure so a retry storm cannot double-send
	n, err := s.Notifications.Insert(ctx, m.Notification)
	if err != nil {
		if m.SendError == nil {
			m.SendError = fmt.Errorf("message was sent, but error inserting record: %w", err)
		}
		return m
	}
	m.Notification = n

	return m
}

// RecipientPreview describes who a notify intent would reach, without sending anything.
type RecipientPreview struct {
	RuleID        string   `json:"rule_id"`
	RuleName      string   `json:"rule_name"`
	RecipientID   int      `json:"recipient_id"`
	RecipientName string   `json:"recipient_name"`
	RecipientType string   `json:"recipient_type"`
	ForwardedFrom []string `json:"forwarded_from,omitempty"`
	Message       string   `json:"message,omitempty"` // rendered body this recipient would get
	Error         string   `json:"error,omitempty"`
}

// PreviewRecipients resolves intents and applies forwards exactly like Send, but delivers nothing
// and stores nothing. Unresolvable intents are reported as previews carrying an error.
func (s *Service) PreviewRecipients(ctx context.Context, t *models.FullTicket, isNew bool, intents []workflow.NotifyIntent) ([]RecipientPreview, error) {
	if t == nil {
		return nil, errors.New("nil ticket received")
	}

	natural := make(recipMap)
	attribution := make(map[int]workflow.NotifyIntent)
	var out []RecipientPreview

	for _, in := range intents {
		recips, err := s.resolveTarget(ctx, t, in.Target)
		if err != nil {
			out = append(out, RecipientPreview{RuleID: in.Rule.RuleID, RuleName: in.Rule.RuleName, Error: err.Error()})
			continue
		}
		for id, r := range recips {
			if _, seen := natural[id]; seen {
				continue
			}
			natural[id] = r
			attribution[id] = in
		}
	}

	for _, m := range s.makeTicketMessages(t, s.applyForwards(ctx, t, natural).toSlice(), isNew, attribution) {
		r := m.WebexRecipient
		rule := attribution[r.origin().ID].Rule
		p := RecipientPreview{
			RuleID:        rule.RuleID,
			RuleName:      rule.RuleName,
			RecipientID:   r.recipient.ID,
			RecipientName: r.recipient.Name,
			RecipientType: string(r.recipient.Type),
			Message:       m.WebexMsg.Markdown,
		}
		for _, f := range r.forwardChain {
			p.ForwardedFrom = append(p.ForwardedFrom, f.Name)
		}
		out = append(out, p)
	}

	return out, nil
}
