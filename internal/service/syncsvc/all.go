package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

func (s *Service) IsSyncing() bool {
	return s.progress.running()
}

// Status is what GET /sync/status returns: whether a sync runs, and the current or last run.
func (s *Service) Status() *models.SyncStatusResponse {
	run := s.progress.snapshot()
	return &models.SyncStatusResponse{Status: run != nil && run.FinishedAt == nil, Run: run}
}

// Start begins a sync in the background and returns once it is registered, or
// ErrSyncRunning while another one is unfinished. startedBy is shown on the Sync page.
// Cancel stops the run, so ctx should not be the request's.
func (s *Service) Start(ctx context.Context, payload *models.SyncPayload, startedBy string) error {
	if payload == nil {
		return errors.New("received nil payload")
	}
	ctx, cancel := context.WithCancel(ctx)
	if err := s.progress.start(payload, startedBy, time.Now(), cancel); err != nil {
		cancel()
		return err
	}
	go s.run(ctx, payload)
	return nil
}

// Cancel stops the running sync, or returns ErrNoSyncRunning. It returns at once; the run
// reports finished on the Sync page once its phases have wound down.
func (s *Service) Cancel(by string) error {
	return s.progress.stop(by, time.Now())
}

func (s *Service) run(ctx context.Context, payload *models.SyncPayload) {
	slog.Info("received sync payload",
		slog.Bool("sync_boards", payload.CWBoards),
		slog.Bool("sync_recipients", payload.WebexRecipients),
		slog.Bool("sync_tickets", payload.CWTickets),
		slog.Any("ticket_board_ids", payload.BoardIDs),
		slog.Int("max_concurrent_syncs", payload.MaxConcurrentSyncs),
	)

	errch := make(chan error, 3)
	var wg sync.WaitGroup

	start := time.Now()
	errored := false
	defer func() {
		s.progress.finish(time.Now())
		if errored {
			slog.Error("sync complete with errors, see logs", "payload", payload, "took_seconds", time.Since(start).Seconds())
		} else {
			slog.Info("sync complete", "payload", payload, "took_seconds", time.Since(start).Seconds())
		}
	}()

	// the boards and recipients syncs write in one transaction each, so a failure or a
	// cancel there saves nothing; the ticket sync writes ticket by ticket
	if payload.CWBoards {
		ph := s.progress.phase(models.SyncPhaseBoards)
		wg.Go(func() {
			err := s.SyncBoards(ctx, ph)
			if err := endPhase(ctx, ph, err, "Rolled back, nothing saved", true); err != nil {
				errch <- fmt.Errorf("syncing connectwise boards: %w", err)
			}
		})
	}

	if payload.WebexRecipients {
		ph := s.progress.phase(models.SyncPhaseWebexRecipients)
		wg.Go(func() {
			err := s.SyncWebexRecipients(ctx, payload.MaxConcurrentSyncs, ph)
			if err := endPhase(ctx, ph, err, "Rolled back, nothing saved", true); err != nil {
				errch <- fmt.Errorf("syncing webex recipients: %w", err)
			}
		})
	}

	if payload.CWTickets {
		ph := s.progress.phase(models.SyncPhaseTickets)
		wg.Go(func() {
			err := s.SyncOpenTickets(ctx, payload.BoardIDs, payload.MaxConcurrentSyncs, ph)
			if err := endPhase(ctx, ph, err, "Stopped", false); err != nil {
				errch <- fmt.Errorf("syncing connectwise tickets: %w", err)
			}
		})
	}

	wg.Wait()
	close(errch)

	for err := range errch {
		if err != nil {
			errored = true
			slog.Error("hook sync", "error", err.Error())
		}
	}
}

// endPhase records how a phase ended and returns err when it is a failure. An error once
// ctx is cancelled is the cancel taking effect, not a failure. A phase that returned nil
// finished before the cancel reached it and stays done.
func endPhase(ctx context.Context, ph *phase, err error, failLabel string, rolledBack bool) error {
	switch {
	case err == nil:
		ph.done()
		return nil
	case ctx.Err() != nil:
		ph.cancelled(rolledBack)
		return nil
	default:
		ph.fail(failLabel, err)
		return err
	}
}

// forEach calls fn for each item, at most limit at a time. Once ctx is done it starts no
// more items and returns ctx's error; it returns nil when every item was started. Either
// way it returns only after every started item has finished.
func forEach[T any](ctx context.Context, items []T, limit int, fn func(T)) error {
	sem := make(chan struct{}, max(limit, 1))
	var wg sync.WaitGroup
	defer wg.Wait()

	for _, item := range items {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case sem <- struct{}{}:
		}
		// both cases can be ready at once and select picks at random
		if err := ctx.Err(); err != nil {
			<-sem
			return err
		}
		wg.Go(func() {
			defer func() { <-sem }()
			fn(item)
		})
	}
	return nil
}
