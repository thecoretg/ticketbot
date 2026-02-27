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
	return s.syncing.Load()
}

func (s *Service) Sync(ctx context.Context, payload *models.SyncPayload) error {
	if payload == nil {
		return errors.New("received nil payload")
	}

	slog.Info("received sync payload",
		slog.Bool("sync_boards", payload.CWBoards),
		slog.Bool("sync_recipients", payload.WebexRecipients),
		slog.Bool("sync_tickets", payload.CWTickets),
		slog.Any("ticket_board_ids", payload.BoardIDs),
		slog.Int("max_concurrent_syncs", payload.MaxConcurrentSyncs),
	)

	if !s.syncing.CompareAndSwap(false, true) {
		return errors.New("sync already in progress")
	}

	errch := make(chan error, 3)
	var wg sync.WaitGroup

	start := time.Now()
	errored := false
	defer func() {
		s.syncing.Store(false)
		if errored {
			slog.Error("sync complete with errors, see logs", "payload", payload, "took_seconds", time.Since(start).Seconds())
		} else {
			slog.Info("sync complete", "payload", payload, "took_seconds", time.Since(start).Seconds())
		}
	}()

	if payload.CWBoards {
		wg.Go(func() {
			if err := s.SyncBoards(ctx); err != nil {
				errch <- fmt.Errorf("syncing connectwise boards: %w", err)
				return
			}
		})
	}

	if payload.WebexRecipients {
		wg.Go(func() {
			if err := s.SyncWebexRecipients(ctx, payload.MaxConcurrentSyncs); err != nil {
				errch <- fmt.Errorf("syncing webex recipients: %w", err)
				return
			}
		})
	}

	if payload.CWTickets {
		wg.Go(func() {
			if err := s.SyncOpenTickets(ctx, payload.BoardIDs, payload.MaxConcurrentSyncs); err != nil {
				errch <- fmt.Errorf("syncing connectwise tickets: %w", err)
				return
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

	return nil
}
