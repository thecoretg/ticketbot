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
func (s *Service) Start(ctx context.Context, payload *models.SyncPayload, startedBy string) error {
	if payload == nil {
		return errors.New("received nil payload")
	}
	if err := s.progress.start(payload, startedBy, time.Now()); err != nil {
		return err
	}
	go s.run(ctx, payload)
	return nil
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

	// the boards and recipients syncs write in one transaction each, so a failure there
	// saves nothing; the ticket sync writes ticket by ticket
	if payload.CWBoards {
		ph := s.progress.phase(models.SyncPhaseBoards)
		wg.Go(func() {
			if err := s.SyncBoards(ctx, ph); err != nil {
				ph.fail("Rolled back, nothing saved", err)
				errch <- fmt.Errorf("syncing connectwise boards: %w", err)
				return
			}
			ph.done()
		})
	}

	if payload.WebexRecipients {
		ph := s.progress.phase(models.SyncPhaseWebexRecipients)
		wg.Go(func() {
			if err := s.SyncWebexRecipients(ctx, payload.MaxConcurrentSyncs, ph); err != nil {
				ph.fail("Rolled back, nothing saved", err)
				errch <- fmt.Errorf("syncing webex recipients: %w", err)
				return
			}
			ph.done()
		})
	}

	if payload.CWTickets {
		ph := s.progress.phase(models.SyncPhaseTickets)
		wg.Go(func() {
			if err := s.SyncOpenTickets(ctx, payload.BoardIDs, payload.MaxConcurrentSyncs, ph); err != nil {
				ph.fail("Stopped", err)
				errch <- fmt.Errorf("syncing connectwise tickets: %w", err)
				return
			}
			ph.done()
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
