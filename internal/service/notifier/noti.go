package notifier

import (
	"context"
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
	Error           error
}

func newRequest(ticket *models.FullTicket) *Request {
	return &Request{
		Ticket:          ticket,
		Notifications:   []models.TicketNotification{},
		MessagesToSend:  []Message{},
		MessagesSent:    []Message{},
		MessagesErrored: []Message{},
		Error:           nil,
	}
}

func (s *Service) Run(ctx context.Context, ticket *models.FullTicket, isNew bool) error {
	if err := s.processNotifications(ctx, ticket, isNew); err != nil {
		return fmt.Errorf("processing notifications: %w", err)
	}

	return nil
}

func (s *Service) processNotifications(ctx context.Context, ticket *models.FullTicket, isNew bool) (err error) {
	req := newRequest(ticket)
	logger := slog.Default().With("ticket_id", ticket.Ticket.ID)
	defer func() {
		logRequest(req, err, logger)
	}()

	rules, err := s.Notifiers.ListByBoard(ctx, ticket.Board.ID)
	if err != nil {
		return fmt.Errorf("listing notifier rules for board: %w", err)
	}
	logger = logger.With(ruleLogGroup(rules))

	if len(rules) == 0 {
		req.NoNotiReason = "no notifier rules found for board"
		return nil
	}

	if isNew {
		var rooms []models.WebexRoom
		for _, n := range rules {
			if !n.NotifyEnabled {
				continue
			}

			r, err := s.Rooms.Get(ctx, n.WebexRoomID)
			if err != nil {
				return fmt.Errorf("getting webex room from notifier rule: %w", err)
			}

			rooms = append(rooms, r)
		}
		req.MessagesToSend = s.makeNewTicketMessages(rooms, ticket)

	} else {

		if ticket.LatestNote == nil {
			req.NoNotiReason = "no note found for ticket"
			return nil
		}

		exists, err := s.checkExistingNoti(ctx, ticket.LatestNote.ID)
		if err != nil {
			return fmt.Errorf("checking for existing notification for ticket note: %w", err)
		}

		if exists {
			req.NoNotiReason = "note already notified"
			return nil
		}

		emails := s.getRecipientEmails(ctx, ticket)
		if len(emails) == 0 {
			req.NoNotiReason = "no resources to notify"
			return nil
		}
		req.MessagesToSend = s.makeUpdatedTicketMessages(ticket, emails)
	}

	if len(req.MessagesToSend) > 0 {
		for _, m := range req.MessagesToSend {
			msg := s.sendNotification(ctx, &m)
			if msg.SendError != nil {
				req.MessagesErrored = append(req.MessagesErrored, *msg)
				continue
			}

			req.MessagesSent = append(req.MessagesSent, *msg)
		}
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

func (s *Service) checkExistingNoti(ctx context.Context, noteID int) (bool, error) {
	exists, err := s.Notifications.ExistsForNote(ctx, noteID)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	return false, nil
}

func (s *Service) sendNotification(ctx context.Context, m *Message) *Message {
	_, err := s.MessageSender.PostMessage(&m.WebexMsg)
	if err != nil {
		m.SendError = fmt.Errorf("sending webex message: %w", err)
	}

	m.Notification, err = s.Notifications.Insert(ctx, m.Notification)
	if err != nil {
		m.SendError = fmt.Errorf("message was sent, but error inserting record: %w", err)
	}

	return m
}

func ruleLogGroup(rules []models.NotifierRule) slog.Attr {
	var attrs []any
	for _, r := range rules {
		g := slog.Group(
			strconv.Itoa(r.ID),
			slog.Int("board_id", r.CwBoardID),
			slog.Int("webex_room_id", r.WebexRoomID),
		)
		attrs = append(attrs, g)
	}

	return slog.Group("notifier_rules", attrs...)
}

func msgsLogGroup(key string, msgs []Message) slog.Attr {
	var msgGrps []any
	msgID := 0
	for _, m := range msgs {
		attrs := []any{
			slog.String("type", m.MsgType),
		}

		if m.WebexRoom != nil {
			g := slog.Group(
				"webex_room",
				slog.Int("id", m.WebexRoom.ID),
				slog.String("name", m.WebexRoom.Name),
				slog.String("type", m.WebexRoom.Type),
			)
			attrs = append(attrs, g)
		}

		if m.ToEmail != nil {
			attrs = append(attrs, slog.String("to_email", *m.ToEmail))
		}

		if m.SendError != nil {
			attrs = append(attrs, slog.String("send_error", m.SendError.Error()))
		}

		msgGrps = append(msgGrps, slog.Group(strconv.Itoa(msgID), attrs...))
		msgID++
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
