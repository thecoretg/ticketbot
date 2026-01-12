package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/pkg/psa"
)

func (s *Service) SyncOpenTickets(ctx context.Context, boardIDs []int, maxSyncs int) error {
	start := time.Now()
	slog.Info("cwsvc: beginning ticket sync", "board_ids", boardIDs)
	defer func() {
		slog.Info("sync: syncing tickets complete", "took_time", time.Since(start))
	}()

	// First phase: get all OPEN tickets from connectwise and ensure they all exist
	// in the database.
	con := "closedFlag = false"
	if len(boardIDs) > 0 {
		con += fmt.Sprintf(" AND %s", boardIDParam(boardIDs))
	}

	params := map[string]string{
		"pageSize":   "100",
		"conditions": con,
	}

	tix, err := s.CW.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from connectwise: %w", err)
	}
	slog.Info("cwsvc: open ticket sync: got open tickets from connectwise", "total_tickets", len(tix))
	sem := make(chan struct{}, maxSyncs)
	var wg sync.WaitGroup
	errCh := make(chan error, len(tix))

	for _, t := range tix {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket psa.Ticket) {
			defer func() { <-sem }()
			defer wg.Done()
			ft, err := s.CW.ProcessTicket(ctx, ticket.ID, "sync")
			if err != nil {
				errCh <- fmt.Errorf("error syncing ticket %d: %w", ticket.ID, err)
				return
			}

			if err := s.Notifier.AddSkippedNotification(ctx, ft, "ticket sync"); err != nil {
				errCh <- fmt.Errorf("skipping notification for ticket %d note %d: %w", ft.Ticket.ID, ft.LatestNote.ID, err)
				return
			}
		}(t)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("sync: syncing open ticket", "error", err.Error())
		}
	}

	return nil
}

func boardIDParam(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	var b strings.Builder
	for i, id := range ids {
		fmt.Fprintf(&b, "board/id = %d", id)
		if i < len(ids)-1 {
			b.WriteString(" OR ")
		}
	}

	return fmt.Sprintf("(%s)", b.String())
}
