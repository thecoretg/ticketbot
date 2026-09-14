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
	natural := make(recipMap)
	attribution := make(map[int]workflow.RuleRef)
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
			attribution[id] = in.Rule
		}
	}

	if len(natural) == 0 {
		logger.Debug("notifier: no recipients to send to")
		return outcomes, nil
	}

	final := s.applyForwards(ctx, t, natural)
	msgs := s.makeTicketMessages(t, final.toSlice(), req.IsNew)

	for _, m := range msgs {
		out := Outcome{
			Rule:      attribution[m.WebexRecipient.origin().ID],
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
