package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/alerts"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg       *models.Config
	ConfigSvc *config.Service
	CW        *cwsvc.Service
	Workflows repos.WorkflowRepository
	Events    repos.TicketEventRepository
	Engine    *workflow.Engine
	Notifier  *notifier.Service
	// Alerter receives the rate-cap alert. nil logs.
	Alerter alerts.Alerter

	locksMu sync.Mutex
	locks   map[int]*ticketLock
	now     func() time.Time
}

// ticketLock serializes intake for one ticket. refs counts waiters so the entry can be dropped
// once nobody holds or wants it, keeping the map from growing with every ticket ever seen.
type ticketLock struct {
	mu   sync.Mutex
	refs int
}

// ProcessOpts describes where an intake request came from and how far it should go.
type ProcessOpts struct {
	Source models.EventSource
	// RunRules is false for bulk syncs, which only refresh storage and record events.
	RunRules bool
}

type Params struct {
	Cfg       *models.Config
	ConfigSvc *config.Service
	CW        *cwsvc.Service
	Workflows repos.WorkflowRepository
	Events    repos.TicketEventRepository
	Engine    *workflow.Engine
	Notifier  *notifier.Service
	Alerter   alerts.Alerter
}

func New(p Params) *Service {
	s := &Service{
		Cfg:       p.Cfg,
		ConfigSvc: p.ConfigSvc,
		CW:        p.CW,
		Workflows: p.Workflows,
		Events:    p.Events,
		Engine:    p.Engine,
		Notifier:  p.Notifier,
		Alerter:   p.Alerter,
		now:       time.Now,
	}
	if s.Alerter == nil {
		s.Alerter = alerts.Log{}
	}
	return s
}

// ProcessTicket ingests one ticket: fetch from ConnectWise, compare with what is stored, run the
// board's workflow (actions may write to ConnectWise), save the post-action ticket, send queued
// notifications, and record every step as ticket events.
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
	defer s.lockTicket(id)()

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

	kind := models.EventUpdated
	if d.IsNew {
		kind = models.EventCreated
	}
	run.add(kind, changePayload(d, f, s.Cfg.NotePreviewLength))

	if guard := s.loopGuard(f, d); guard != nil {
		run.add(models.EventLoopGuard, guard)
		logger.Debug("ticketbot: loop guard tripped", "reason", guard.Reason)
		_, err := s.CW.SaveTicket(ctx, f, string(opts.Source))
		return err
	}

	var res *workflow.Result
	if opts.RunRules {
		res = s.runWorkflow(ctx, run, f, d)
		if res != nil {
			f = &cwsvc.Fetched{Ticket: res.Ticket, Note: res.LatestNote, TriggerNote: res.TriggerNote}
		}
	}

	ft, err := s.CW.SaveTicket(ctx, f, string(opts.Source))
	if err != nil {
		run.add(models.EventError, models.ErrorPayload{Stage: "save", Error: err.Error()})
		return fmt.Errorf("saving ticket %d: %w", id, err)
	}

	if res != nil && len(res.Notifies) > 0 {
		s.sendNotifications(ctx, run, ft, d.IsNew, res)
	}

	return nil
}

// loopGuard reports why rules must not run for a change ticketbot made itself, or nil. It reads
// the record, never the webhook payload: a ConnectWise callback's MemberID names the API member
// that owns the callback, not whoever changed the ticket, so it is the same for every delivery.
func (s *Service) loopGuard(f *cwsvc.Fetched, d decision) *models.LoopGuardPayload {
	apiID := strings.TrimSpace(s.Cfg.CWAPIMemberIdentifier)
	if apiID == "" {
		return nil
	}

	switch {
	case d.NewNote && f.Note != nil && strings.EqualFold(f.Note.Member.Identifier, apiID):
		return &models.LoopGuardPayload{Reason: "note_author", Identifier: apiID}
	case len(d.Changes) > 0 && !d.NewNote && strings.EqualFold(f.Ticket.Info.UpdatedBy, apiID):
		return &models.LoopGuardPayload{Reason: "updated_by", Identifier: apiID}
	}

	return nil
}

// runWorkflow looks up the board's workflow and runs it, recording workflow and action events.
// It returns nil when there is no runnable workflow.
func (s *Service) runWorkflow(ctx context.Context, run *run, f *cwsvc.Fetched, d decision) *workflow.Result {
	wf, err := s.Workflows.GetByBoard(ctx, f.Ticket.Board.ID)
	if err != nil {
		if !errors.Is(err, models.ErrWorkflowNotFound) {
			run.add(models.EventError, models.ErrorPayload{Stage: "workflow", Error: err.Error()})
			return nil
		}
		run.add(models.EventWorkflow, models.WorkflowPayload{Found: false})
		return nil
	}

	if !wf.Enabled {
		run.add(models.EventWorkflow, models.WorkflowPayload{WorkflowID: &wf.ID, WorkflowName: wf.Name, Found: true, Enabled: false})
		return nil
	}

	run.dryRun = s.Cfg.MasterDryRun || wf.DryRun

	res, err := s.Engine.Run(ctx, wf, workflow.Input{
		Ticket:      f.Ticket,
		TriggerNote: f.Note,
		IsNew:       d.IsNew,
		NewNote:     d.NewNote,
		Changes:     d.Changes,
		DryRun:      run.dryRun,
	})
	if err != nil {
		run.add(models.EventError, models.ErrorPayload{Stage: "workflow", Error: err.Error()})
		return nil
	}

	run.add(models.EventWorkflow, workflowPayload(wf, res))
	for _, a := range res.Actions {
		run.add(models.EventAction, a.Payload())
	}
	if res.RateCapped {
		msg := fmt.Sprintf("ticket %d reached the write cap (%d writes in 15 minutes); its ConnectWise writes are blocked for an hour", f.Ticket.ID, s.Cfg.WriteCapPerTicket)
		run.add(models.EventError, models.ErrorPayload{Stage: "rate_cap", Error: msg})
		s.Alerter.Alert(ctx, "Write cap reached", msg+fmt.Sprintf("\nWorkflow: %s. Check it for a loop before the block lifts.", wf.Name))
	}

	if res.LearnedAPIMember != "" && strings.TrimSpace(s.Cfg.CWAPIMemberIdentifier) == "" && s.ConfigSvc != nil {
		id := res.LearnedAPIMember
		if _, err := s.ConfigSvc.Update(ctx, &models.ConfigUpdateParams{CWAPIMemberIdentifier: &id}); err != nil {
			slog.Error("ticketbot: saving learned api member identifier", "identifier", id, "error", err.Error())
		} else {
			slog.Info("ticketbot: learned connectwise api member identifier", "identifier", id)
		}
	}

	return res
}

func (s *Service) sendNotifications(ctx context.Context, run *run, ft *models.FullTicket, isNew bool, res *workflow.Result) {
	outs, err := s.Notifier.Send(ctx, notifier.SendRequest{
		Ticket:  ft,
		IsNew:   isNew,
		DryRun:  run.dryRun,
		Intents: res.Notifies,
	})
	if err != nil {
		run.add(models.EventError, models.ErrorPayload{Stage: "notify", Error: err.Error()})
		return
	}

	for _, o := range outs {
		run.add(models.EventNotification, notificationPayload(o))
	}
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
	defer s.lockTicket(id)()

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

// lockTicket acquires the per-ticket lock and returns the release function.
func (s *Service) lockTicket(id int) func() {
	s.locksMu.Lock()
	if s.locks == nil {
		s.locks = make(map[int]*ticketLock)
	}
	l := s.locks[id]
	if l == nil {
		l = &ticketLock{}
		s.locks[id] = l
	}
	l.refs++
	s.locksMu.Unlock()

	l.mu.Lock()
	return func() {
		l.mu.Unlock()
		s.locksMu.Lock()
		l.refs--
		if l.refs == 0 {
			delete(s.locks, id)
		}
		s.locksMu.Unlock()
	}
}
