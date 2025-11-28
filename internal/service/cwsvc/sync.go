package cwsvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func (s *Service) SyncOpenTickets(ctx context.Context, boardIDs []int, maxSyncs int) error {
	start := time.Now()
	slog.Info("cwsvc: beginning ticket sync", "board_ids", boardIDs)
	con := "closedFlag = false"
	if len(boardIDs) > 0 {
		con += fmt.Sprintf(" AND %s", boardIDParam(boardIDs))
	}

	params := map[string]string{
		"pageSize":   "100",
		"conditions": con,
	}

	tix, err := s.cwClient.ListTickets(params)
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
			if _, err := s.ProcessTicket(ctx, t.ID); err != nil {
				slog.Error("cwsvc: ticket sync error", "ticket_id", t.ID, "error", err)
				errCh <- fmt.Errorf("error syncing ticket %d: %w", t.ID, err)
			} else {
				errCh <- nil
			}
		}(t)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("cwsvc: syncing ticket", "error", err)
		}
	}
	slog.Info("cwsvc: syncing tickets complete", "took_time", time.Since(start))
	return nil
}

func (s *Service) SyncBoards(ctx context.Context) error {
	// TODO: make this less bad

	start := time.Now()
	slog.Info("beginning connectwise board sync")
	cwb, err := s.cwClient.ListBoards(nil)
	if err != nil {
		return fmt.Errorf("listing connectwise boards: %w", err)
	}
	slog.Info("board sync: got boards from connectwise", "total_boards", len(cwb))

	sb, err := s.Boards.List(ctx)
	if err != nil {
		return fmt.Errorf("listing boards from store: %w", err)
	}
	slog.Info("board sync: got boards from store", "total_boards", len(sb))

	//TODO: this is a bandaid. Move this logic to the repo.
	txSvc := s
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning tx: %w", err)
		}

		txSvc = s.withTX(tx)

		defer func() {
			_ = tx.Rollback(ctx)
		}()
	}

	for _, b := range txSvc.boardsToUpsert(cwb) {
		if _, err := txSvc.Boards.Upsert(ctx, b); err != nil {
			return fmt.Errorf("upserting board %d (%s): %w", b.ID, b.Name, err)
		}
	}

	for _, b := range txSvc.boardsToDelete(cwb, sb) {
		if err := txSvc.Boards.Delete(ctx, b.ID); err != nil {
			return fmt.Errorf("deleting board %d (%s): %w", b.ID, b.Name, err)
		}
	}

	//TODO: this is a bandaid. Move this logic to the repo.
	if s.pool != nil {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing tx: %w", err)
		}
	}

	slog.Info("board sync complete", "took_time", time.Since(start).Seconds())
	return nil
}

func (s *Service) boardsToUpsert(cwBoards []psa.Board) []models.Board {
	var toUpsert []models.Board
	for _, c := range cwBoards {
		b := models.Board{
			ID:   c.ID,
			Name: c.Name,
		}
		toUpsert = append(toUpsert, b)
	}

	return toUpsert
}

func (s *Service) boardsToDelete(cwBoards []psa.Board, storeBoards []models.Board) []models.Board {
	ci := make(map[int]psa.Board)
	for _, c := range cwBoards {
		ci[c.ID] = c
	}

	var toDelete []models.Board
	for _, b := range storeBoards {
		if _, ok := ci[b.ID]; !ok {
			toDelete = append(toDelete, b)
		}
	}

	return toDelete
}

func boardIDParam(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	param := ""
	for i, id := range ids {
		param += fmt.Sprintf("board/id = %d", id)
		if i < len(ids)-1 {
			param += " OR "
		}
	}

	return fmt.Sprintf("(%s)", param)
}
