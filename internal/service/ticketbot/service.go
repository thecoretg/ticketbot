package ticketbot

import (
	"context"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
)

type Service struct {
	Cfg      models.Config
	CW       *cwsvc.Service
	Notifier *notifier.Service
}

func New(cfg models.Config, cw *cwsvc.Service, ns *notifier.Service) *Service {
	return &Service{
		Cfg:      cfg,
		CW:       cw,
		Notifier: ns,
	}
}

func (s *Service) ProcessNewTicket(ctx context.Context, id int) ([]models.TicketNotification, error) {
	ticket, err := s.CW.ProcessTicket(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("processing ticket: %w", err)
	}

	var notis []models.TicketNotification
	if s.Cfg.AttemptNotify {
		res := s.Notifier.ProcessWithNewTicket(ctx, ticket)
		if res.Error != nil {
			return res.Notifications, fmt.Errorf("processing notifications: %w", res.Error)
		}

		notis = res.Notifications
	}

	return notis, nil
}
