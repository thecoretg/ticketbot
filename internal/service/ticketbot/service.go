package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
)

type Service struct {
	Cfg         *models.Config
	CW          *cwsvc.Service
	Notifier    *notifier.Service
	ticketLocks sync.Map
}

func New(cfg *models.Config, cw *cwsvc.Service, ns *notifier.Service) *Service {
	return &Service{
		Cfg:      cfg,
		CW:       cw,
		Notifier: ns,
	}
}

func (s *Service) ProcessTicket(ctx context.Context, id int) error {
	start := time.Now()
	slog.Debug("ticketbot: request received", "ticket_id", id)

	defer func() {
		took := time.Since(start).Seconds()
		slog.Debug("ticketbot: request finished", "ticket_id", id, "took_seconds", took)
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
	isNew := !exists

	ticket, err := s.CW.ProcessTicket(ctx, id)
	if err != nil {
		return fmt.Errorf("processing ticket %d: %w", id, err)
	}

	if s.Cfg.AttemptNotify {
		if err := s.Notifier.Run(ctx, ticket, isNew); err != nil {
			return fmt.Errorf("running notifier for ticket %d: %w", id, err)
		}
		return nil
	}

	slog.Debug("ticketbot: attempt notify disabled", "ticket_id", id)
	return nil
}

func (s *Service) getTicketLock(id int) *sync.Mutex {
	li, _ := s.ticketLocks.LoadOrStore(id, &sync.Mutex{})
	return li.(*sync.Mutex)
}
