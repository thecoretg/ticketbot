package ticketbot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
)

type Service struct {
	Cfg      *models.Config
	CW       *cwsvc.Service
	Notifier *notifier.Service
}

func New(cfg *models.Config, cw *cwsvc.Service, ns *notifier.Service) *Service {
	return &Service{
		Cfg:      cfg,
		CW:       cw,
		Notifier: ns,
	}
}

func (s *Service) ProcessTicket(ctx context.Context, id int, isNew bool) error {
	ticket, err := s.CW.ProcessTicket(ctx, id)
	if err != nil {
		return fmt.Errorf("processing ticket: %w", err)
	}

	switch s.Cfg.AttemptNotify {
	case true:
		slog.Debug("ticketbot: attempt notify enabled", "ticket_id", id)
		res := s.Notifier.ProcessTicket(ctx, ticket, isNew)
		if res.Error != nil {
			return fmt.Errorf("processing notifications: %w", res.Error)
		}

		if res.NoNotiReason != "" {
			slog.Info("ticketbot: notification not sent", "ticket_id", id, "reason", res.NoNotiReason)
			return nil
		}

		for _, m := range res.MessagesToSend {
			slog.Info("ticketbot: notification sent", "ticket_id", id, "recipients", getSentTo(m))
		}

		for _, m := range res.MessagesErrored {
			if m.SendError != nil {
				slog.Error("ticketbot: error sending notification", "ticket_id", id, "recipients", getSentTo(m), "error", m.SendError)
			}
		}

		return nil

	case false:
		slog.Debug("ticketbot: attempt notify disabled", "ticket_id", id)
	}

	return nil
}

func getSentTo(n notifier.Message) string {
	sentTo := ""
	if n.ToEmail != nil {
		sentTo = *n.ToEmail
	} else if n.WebexRoom != nil {
		sentTo = n.WebexRoom.Name
	}

	return sentTo
}
