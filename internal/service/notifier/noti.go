package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/thecoretg/ticketbot/internal/models"
)

type Request struct {
	Ticket          *models.FullTicket
	Notifications   []models.TicketNotification
	MessagesToSend  []Message
	MessagesSent    []Message
	MessagesErrored []Message
	NoNotiReason    string
}

func newRequest(ticket *models.FullTicket) *Request {
	return &Request{
		Ticket: ticket,
	}
}

func (s *Service) Run(ctx context.Context, t *models.FullTicket, isNew bool) error {
	return s.processNotifications(ctx, t, isNew)
}

func (s *Service) AddSkippedNotification(ctx context.Context, t *models.FullTicket, source string) error {
	if t == nil {
		return errors.New("received nil ticket")
	}

	if t.LatestNote != nil && t.LatestNote.ID != 0 {
		ti := t.Ticket.ID
		ni := t.LatestNote.ID

		// Check if notification was already sent/skipped for this note
		exists, err := s.Notifications.ExistsForNote(ctx, ni)
		if err != nil {
			return fmt.Errorf("checking if notification exists for note: %w", err)
		}

		if exists {
			slog.Debug("notification skipper: notification already exists", "ticket_id", ti, "note_id", ni)
			return nil
		}

		n := models.TicketNotification{
			TicketID:     ti,
			TicketNoteID: &ni,
			Skipped:      true,
		}

		n, err = s.Notifications.Insert(ctx, n)
		if err != nil {
			return fmt.Errorf("inserting notification: %w", err)
		}

		slog.Debug("notification skipper: inserted skipped notification", "source", source, "ticket_id", ti, "note_id", ni, "notification_id", n.ID)
		return nil
	}

	slog.Debug("notification skipper: no note to notify", "ticket_id", t.Ticket.ID)
	return nil
}

func (s *Service) processNotifications(ctx context.Context, t *models.FullTicket, isNew bool) (err error) {
	if t == nil {
		return errors.New("nil ticket received")
	}

	req := newRequest(t)
	logger := slog.Default().With("ticket_id", t.Ticket.ID)
	defer func() {
		logRequest(req, err, logger)
	}()

	rules, err := s.NotifierRules.ListByBoard(ctx, t.Board.ID)
	if err != nil {
		return fmt.Errorf("listing notifier rules for board: %w", err)
	}
	logger = logger.With(ruleLogGroup(rules))

	rules = filterActiveRules(rules)
	if len(rules) == 0 {
		req.NoNotiReason = "no notifier rules found for board"
		return nil
	}

	if !isNew {
		if t.LatestNote == nil {
			req.NoNotiReason = "no note found for ticket"
			return nil
		}

		exists, err := s.Notifications.ExistsForNote(ctx, t.LatestNote.ID)
		if err != nil {
			return fmt.Errorf("checking for existing notification for ticket note: %w", err)
		}

		if exists {
			req.NoNotiReason = "note already notified"
			return nil
		}
	}

	recips, err := s.getAllRecipients(ctx, t, rules, isNew)
	if err != nil {
		return fmt.Errorf("getting recipients: %w", err)
	}

	if len(recips) == 0 {
		req.NoNotiReason = "no recipients to send to"
		return nil
	}

	req.MessagesToSend = s.makeTicketMessages(t, recips, isNew)

	for _, m := range req.MessagesToSend {
		msg := s.sendNotification(ctx, &m)
		if msg.SendError != nil {
			req.MessagesErrored = append(req.MessagesErrored, *msg)
			continue
		}

		req.MessagesSent = append(req.MessagesSent, *msg)
	}

	if len(req.MessagesSent) > 0 {
		logger = logger.With(msgsLogGroup("messages_sent", req.MessagesSent))
	}

	if len(req.MessagesErrored) > 0 {
		logger = logger.With(msgsLogGroup("messages_errored", req.MessagesErrored))
		return fmt.Errorf("errors occurred sending %d messages; see logs for details", len(req.MessagesErrored))
	}

	return nil
}

func (s *Service) sendNotification(ctx context.Context, m *Message) *Message {
	n := m.Notification
	logger := slog.Default().With(
		slog.Int("ticket_id", n.TicketID),
		slog.Int("ticket_note_id", ptrToInt(n.TicketNoteID)),
		slog.String("recipient", m.WebexRecipient.recipient.Name),
	)

	logger.Debug("notifier: sending notification")
	_, err := s.MessageSender.PostMessage(&m.WebexMsg)
	if err != nil {
		m.SendError = fmt.Errorf("sending webex message: %w", err)
	}

	logger.Debug("inserting notification into store")
	m.Notification, err = s.Notifications.Insert(ctx, m.Notification)
	if err != nil {
		if m.SendError == nil {
			m.SendError = fmt.Errorf("message was sent, but error inserting record: %w", err)
		}
	}

	return m
}

func filterActiveRules(rules []models.NotifierRule) []models.NotifierRule {
	var active []models.NotifierRule
	for _, r := range rules {
		if r.NotifyEnabled {
			active = append(active, r)
		}
	}

	return active
}

func ruleLogGroup(rules []models.NotifierRule) slog.Attr {
	var attrs []any
	for _, r := range rules {
		g := slog.Group(
			strconv.Itoa(r.ID),
			slog.Int("board_id", r.CwBoardID),
			slog.Int("webex_recipient_id", r.WebexRecipientID),
		)
		attrs = append(attrs, g)
	}

	return slog.Group("notifier_rules", attrs...)
}

func msgsLogGroup(key string, msgs []Message) slog.Attr {
	var msgGrps []any
	for i, m := range msgs {
		attrs := []any{
			slog.String("type", m.MsgType),
		}

		// TODO: add logging for if it was a forward

		if m.WebexRecipient.recipient.ID != 0 {
			g := slog.Group(
				"webex_recipient",
				slog.Int("id", m.WebexRecipient.recipient.ID),
				slog.String("name", m.WebexRecipient.recipient.Name),
				slog.String("type", string(m.WebexRecipient.recipient.Name)),
			)
			attrs = append(attrs, g)
		}

		if m.SendError != nil {
			attrs = append(attrs, slog.String("send_error", m.SendError.Error()))
		}

		msgGrps = append(msgGrps, slog.Group(strconv.Itoa(i), attrs...))
	}

	return slog.Group(key, msgGrps...)
}

func logRequest(req *Request, err error, logger *slog.Logger) {
	if req.NoNotiReason != "" {
		logger = logger.With("no_noti_reason", req.NoNotiReason)
	}

	if err != nil {
		logger.Error("error occured with notification", "error", err)
	} else {
		logger.Info("notification processed")
	}
}

func ptrToInt(p *int) int {
	i := 0
	if p != nil {
		i = *p
	}

	return i
}
