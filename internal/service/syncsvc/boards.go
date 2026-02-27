package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/psa"
)

func (s *Service) SyncBoards(ctx context.Context) error {
	start := time.Now()
	slog.Info("beginning connectwise board sync")
	cwb, err := s.CW.CWClient.ListBoards(nil)
	if err != nil {
		return fmt.Errorf("listing connectwise boards: %w", err)
	}
	slog.Info("board sync: got boards from connectwise", "total_boards", len(cwb))

	sb, err := s.CW.Boards.List(ctx)
	if err != nil {
		return fmt.Errorf("listing boards from store: %w", err)
	}
	slog.Info("board sync: got boards from store", "total_boards", len(sb))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}

	txSvc := s.withTx(tx)
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for _, b := range boardsToUpsert(cwb) {
		if _, err := txSvc.CW.Boards.Upsert(ctx, b); err != nil {
			slog.Error("board sync: upserting board", "board_id", b.ID, "error", err.Error())
			continue
		}

		if err := txSvc.SyncBoardStatuses(ctx, b.ID); err != nil {
			slog.Error("board sync: status sync", "board_id", b.ID, "error", err.Error())
		}
	}

	for _, b := range boardsToDelete(cwb, sb) {
		if err := txSvc.CW.Boards.SoftDelete(ctx, b.ID); err != nil {
			return fmt.Errorf("soft deleting board %d (%s): %w", b.ID, b.Name, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	slog.Info("board sync complete", "took_time", time.Since(start).Seconds())
	return nil
}

func boardsToUpsert(cwBoards []psa.Board) []*models.Board {
	var toUpsert []*models.Board
	for _, c := range cwBoards {
		b := &models.Board{
			ID:   c.ID,
			Name: c.Name,
		}
		toUpsert = append(toUpsert, b)
	}

	return toUpsert
}

func boardsToDelete(cwBoards []psa.Board, storeBoards []*models.Board) []*models.Board {
	ci := make(map[int]psa.Board)
	for _, c := range cwBoards {
		ci[c.ID] = c
	}

	var toDelete []*models.Board
	for _, b := range storeBoards {
		// skip soft deleted boards
		if b.Deleted {
			continue
		}
		if _, ok := ci[b.ID]; !ok {
			toDelete = append(toDelete, b)
		}
	}

	return toDelete
}
