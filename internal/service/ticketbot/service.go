package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg         *models.Config
	CW          *cwsvc.Service
	Notifier    *notifier.Service
	Events      repos.TicketEventRepository
	ticketLocks sync.Map
	now         func() time.Time
}

// ProcessOpts describes where an intake request came from and how far it should go.
type ProcessOpts struct {
	Source models.EventSource
	// WebhookMemberID is the ConnectWise member identifier that made the change, when known.
	WebhookMemberID string
	// RunRules is false for bulk syncs, which only refresh storage and record events.
	RunRules bool
}

func New(cfg *models.Config, cw *cwsvc.Service, ns *notifier.Service, events repos.TicketEventRepository) *Service {
	return &Service{
		Cfg:      cfg,
		CW:       cw,
		Notifier: ns,
		Events:   events,
		now:      time.Now,
	}
}

// ProcessTicket ingests one ticket: fetch from ConnectWise, compare with what is stored, save,
// record events, and (for now) run the legacy notifier when something changed.
func (s *Service) ProcessTicket(ctx context.Context, id int, opts ProcessOpts) (err error) {
	start := s.now()
	logger := slog.Default().With("ticket_id", id, "source", opts.Source)
	logger.Debug("ticketbot: request received")

	defer func() {
		took := time.Since(start).Seconds()
		if err != nil {
			logger.Error("ticketbot: request finished with error", "took_seconds", took, "error", err.Error())
			return
		}
		logger.Debug("ticketbot: request finished", "took_seconds", took)
	}()

	// Prevent a ticket from processing multiple times to prevent duplicate notifications.
	// Connectwise frequently sends multiple hooks for the same ticket simultaneously.
	lock := s.getTicketLock(id)
	lock.Lock()
	defer lock.Unlock()

	run := newRun(id, opts.Source, s.now)
	run.dryRun = s.Cfg.MasterDryRun
	defer run.flush(ctx, s.Events)

	stored, err := s.CW.Tickets.Get(ctx, id)
	if err != nil && !errors.Is(err, models.ErrTicketNotFound) {
		return fmt.Errorf("getting stored ticket %d: %w", id, err)
	}

	f, err := s.CW.FetchTicket(ctx, id)
	if err != nil {
		if errors.Is(err, cwsvc.ErrTicketWasDeleted) {
			return s.handleDeleted(ctx, run, stored)
		}
		run.add(models.EventError, models.ErrorPayload{Stage: "fetch", Error: err.Error()})
		return fmt.Errorf("fetching ticket %d: %w", id, err)
	}

	d := decide(stored, f)
	if !d.Changed() {
		// Nothing to record; refresh the stored copy quietly so raw JSON stays current.
		if _, err := s.CW.SaveTicket(ctx, f, string(opts.Source)); err != nil {
			return fmt.Errorf("saving ticket %d: %w", id, err)
		}
		logger.Debug("ticketbot: no changes detected")
		return nil
	}

	ft, err := s.CW.SaveTicket(ctx, f, string(opts.Source))
	if err != nil {
		run.add(models.EventError, models.ErrorPayload{Stage: "save", Error: err.Error()})
		return fmt.Errorf("saving ticket %d: %w", id, err)
	}

	kind := models.EventUpdated
	if d.IsNew {
		kind = models.EventCreated
	}
	run.add(kind, changePayload(d, f))

	if !opts.RunRules {
		return nil
	}

	if s.Cfg.MasterDryRun {
		logger.Debug("ticketbot: master dry run enabled; skipping notifications")
		return nil
	}

	if err := s.Notifier.Run(ctx, ft, d.IsNew); err != nil {
		run.add(models.EventError, models.ErrorPayload{Stage: "notify", Error: err.Error()})
		return fmt.Errorf("running notifier for ticket %d: %w", id, err)
	}

	return nil
}

func (s *Service) handleDeleted(ctx context.Context, run *run, stored *models.Ticket) error {
	if stored == nil {
		slog.Debug("ticketbot: ticket deleted from connectwise and never stored", "ticket_id", run.ticketID)
		return nil
	}

	if err := s.CW.SoftDeleteTicket(ctx, stored.ID); err != nil {
		return fmt.Errorf("soft deleting ticket %d: %w", stored.ID, err)
	}

	if !stored.Deleted {
		run.add(models.EventDeleted, nil)
	}

	return nil
}

// SoftDeleteTicket marks a ticket deleted and records the event.
func (s *Service) SoftDeleteTicket(ctx context.Context, id int, source models.EventSource) error {
	lock := s.getTicketLock(id)
	lock.Lock()
	defer lock.Unlock()

	stored, err := s.CW.Tickets.Get(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			return nil
		}
		return fmt.Errorf("getting stored ticket %d: %w", id, err)
	}

	run := newRun(id, source, s.now)
	defer run.flush(ctx, s.Events)

	return s.handleDeleted(ctx, run, stored)
}

func (s *Service) getTicketLock(id int) *sync.Mutex {
	li, _ := s.ticketLocks.LoadOrStore(id, &sync.Mutex{})
	return li.(*sync.Mutex)
}
