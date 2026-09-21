// Package intake is the persisted webhook queue. A ConnectWise callback is written to the
// webhook_intake table and acknowledged at once; workers drain the table, retrying transient
// failures across restarts, and park what keeps failing for an admin to retry or discard.
package intake

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/alerts"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/models"
)

// Processor is the part of ticketbot.Service the worker drives.
type Processor interface {
	ProcessTicket(ctx context.Context, id int, opts ticketbot.ProcessOpts) error
	SoftDeleteTicket(ctx context.Context, id int, source models.EventSource) error
}

// Alerter receives operator alerts. The default logs at error level; the ops room replaces it.
type Alerter = alerts.Alerter

// RetentionConfig is the slice of app config the purge and staleness goroutines read.
type RetentionConfig interface {
	GetLogRetentionDays() int
	GetStaleAlertMinutes() int
}

// DefaultBackoff paces retries. Six attempts spread over about two hours cover a ConnectWise or
// database outage of normal length; anything longer is parked as failed.
var DefaultBackoff = []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute, time.Hour}

const (
	// workers is how many rows process at once. The claim query never hands two workers the same
	// ticket, so this only bounds concurrent ConnectWise fetches for different tickets.
	workers = 4
	// pollInterval is the fallback wake-up for rows whose retry time has arrived and rows inserted
	// by another process. Enqueue wakes a worker immediately, so this is rarely the trigger.
	pollInterval = 2 * time.Second
	// processTimeout bounds one attempt so a hung ConnectWise call cannot pin a worker forever.
	processTimeout = 5 * time.Minute
	purgeInterval  = time.Hour
)

type Service struct {
	Repo      repos.WebhookIntakeRepository
	Processor Processor
	Alerter   Alerter
	Cfg       RetentionConfig
	Backoff   []time.Duration

	wake chan struct{}
	wg   sync.WaitGroup
	now  func() time.Time
}

type Params struct {
	Repo      repos.WebhookIntakeRepository
	Processor Processor
	Alerter   Alerter
	Cfg       RetentionConfig
}

func New(p Params) *Service {
	s := &Service{
		Repo:      p.Repo,
		Processor: p.Processor,
		Alerter:   p.Alerter,
		Cfg:       p.Cfg,
		Backoff:   DefaultBackoff,
		wake:      make(chan struct{}, 1),
		now:       time.Now,
	}
	if s.Alerter == nil {
		s.Alerter = alerts.Log{}
	}
	return s
}

// Enqueue persists one webhook and wakes a worker. It is the only thing the HTTP handler does.
func (s *Service) Enqueue(ctx context.Context, ticketID int, action models.IntakeAction, payload []byte) (*models.WebhookIntake, error) {
	row, err := s.Repo.Insert(ctx, ticketID, action, payload)
	if err != nil {
		return nil, fmt.Errorf("queueing webhook for ticket %d: %w", ticketID, err)
	}
	s.signal()
	return row, nil
}

// Start returns rows a previous process left in flight to pending, then launches the workers and
// the purge loop. They stop when ctx is cancelled; call Wait to let an in-flight row finish.
func (s *Service) Start(ctx context.Context) {
	if n, err := s.Repo.ResetProcessing(ctx); err != nil {
		slog.Error("intake: resetting in-flight rows", "error", err.Error())
	} else if n > 0 {
		slog.Info("intake: re-queued rows left in flight by the previous process", "count", n)
	}

	for i := 0; i < workers; i++ {
		s.wg.Add(1)
		go s.runWorker(ctx)
	}
	s.wg.Add(2)
	go s.runPurge(ctx)
	go s.runStaleWatch(ctx)
}

// Wait blocks until every worker has exited or ctx is done.
func (s *Service) Wait(ctx context.Context) {
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
		slog.Warn("intake: shutdown timed out with rows still in flight; they will be re-queued on start")
	}
}

// Retry re-queues a failed row. Discard drops it. Both are admin actions.
func (s *Service) Retry(ctx context.Context, id int64) (*models.WebhookIntake, error) {
	row, err := s.Repo.Retry(ctx, id)
	if err != nil {
		return nil, err
	}
	s.signal()
	return row, nil
}

func (s *Service) Discard(ctx context.Context, id int64) (*models.WebhookIntake, error) {
	return s.Repo.Discard(ctx, id)
}

func (s *Service) List(ctx context.Context, status *models.IntakeStatus, limit int) ([]*models.WebhookIntake, error) {
	return s.Repo.List(ctx, status, limit)
}

func (s *Service) Stats(ctx context.Context) (*models.IntakeStats, error) {
	return s.Repo.Stats(ctx)
}

// Hourly returns webhook counts per hour for the last days days, oldest first, for the dashboard.
func (s *Service) Hourly(ctx context.Context, days int) ([]models.IntakeHourCount, error) {
	if days <= 0 || days > 30 {
		days = 7
	}
	return s.Repo.CountsByHour(ctx, s.now().Add(-time.Duration(days)*24*time.Hour).Truncate(time.Hour))
}

func (s *Service) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func (s *Service) runWorker(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		if ctx.Err() != nil {
			return
		}
		row, err := s.Repo.Claim(ctx)
		switch {
		case err == nil:
			s.process(ctx, row)
			// Another row may be ready behind this one; loop without waiting.
			continue
		case errors.Is(err, models.ErrIntakeEmpty):
		case ctx.Err() != nil:
			return
		default:
			slog.Error("intake: claiming next webhook", "error", err.Error())
		}

		select {
		case <-s.wake:
		case <-ticker.C:
		case <-ctx.Done():
			return
		}
	}
}

// process runs one attempt. The attempt itself ignores shutdown cancellation so a half-finished
// ticket write is never abandoned mid-way; the timeout is the only thing that stops it.
func (s *Service) process(ctx context.Context, row *models.WebhookIntake) {
	attemptCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), processTimeout)
	defer cancel()

	err := s.run(attemptCtx, row)
	if err == nil {
		if ferr := s.Repo.Finish(attemptCtx, row.ID); ferr != nil {
			slog.Error("intake: marking webhook done", "id", row.ID, "error", ferr.Error())
		}
		return
	}

	logger := slog.Default().With("id", row.ID, "ticket_id", row.TicketID, "action", row.Action, "attempt", row.Attempts)
	if row.Attempts <= len(s.Backoff) {
		delay := s.Backoff[row.Attempts-1]
		logger.Warn("intake: processing webhook failed; retrying", "retry_in", delay.String(), "error", err.Error())
		if rerr := s.Repo.Reschedule(attemptCtx, row.ID, s.now().Add(delay), err.Error()); rerr != nil {
			logger.Error("intake: rescheduling webhook", "error", rerr.Error())
		}
		return
	}

	logger.Error("intake: processing webhook failed; giving up", "error", err.Error())
	if ferr := s.Repo.Fail(attemptCtx, row.ID, err.Error()); ferr != nil {
		logger.Error("intake: marking webhook failed", "error", ferr.Error())
	}
	s.Alerter.Alert(attemptCtx, fmt.Sprintf("Webhook for ticket %d failed after %d attempts", row.TicketID, row.Attempts),
		fmt.Sprintf("Last error: %s\nRetry or discard it from the Intake page.", err.Error()))
}

func (s *Service) run(ctx context.Context, row *models.WebhookIntake) error {
	switch row.Action {
	case models.IntakeAdded, models.IntakeUpdated:
		return s.Processor.ProcessTicket(ctx, row.TicketID, ticketbot.ProcessOpts{Source: models.SourceWebhook, RunRules: true})
	case models.IntakeDeleted:
		return s.Processor.SoftDeleteTicket(ctx, row.TicketID, models.SourceWebhook)
	default:
		return fmt.Errorf("unknown webhook action %q", row.Action)
	}
}

func (s *Service) runPurge(ctx context.Context) {
	defer s.wg.Done()
	ticker := time.NewTicker(purgeInterval)
	defer ticker.Stop()

	purge := func() {
		days := 7
		if s.Cfg != nil && s.Cfg.GetLogRetentionDays() > 0 {
			days = s.Cfg.GetLogRetentionDays()
		}
		n, err := s.Repo.DeleteFinishedBefore(ctx, s.now().AddDate(0, 0, -days))
		if err != nil {
			if ctx.Err() == nil {
				slog.Warn("intake: purging finished rows", "error", err.Error())
			}
			return
		}
		if n > 0 {
			slog.Info("intake: purged finished rows", "deleted", n, "retention_days", days)
		}
	}

	purge()
	for {
		select {
		case <-ticker.C:
			purge()
		case <-ctx.Done():
			return
		}
	}
}
