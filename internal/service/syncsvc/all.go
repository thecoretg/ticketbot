package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (s *Service) Sync(ctx context.Context, payload *models.SyncPayload) error {
	if payload == nil {
		return errors.New("received nil payload")
	}

	errch := make(chan error, 3)
	var wg sync.WaitGroup

	start := time.Now()
	errored := false
	defer func() {
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
			slog.Error("hook sync", "error", err)
		}
	}

	return nil
}
