package cwsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func (s *Service) ListBoards(ctx context.Context) ([]models.Board, error) {
	return s.Boards.List(ctx)
}

func (s *Service) GetBoard(ctx context.Context, id int) (models.Board, error) {
	return s.Boards.Get(ctx, id)
}

func (s *Service) SyncBoards(ctx context.Context) error {
	start := time.Now()
	slog.Debug("beginning connectwise board sync")
	cwb, err := s.cwClient.ListBoards(nil)
	if err != nil {
		return fmt.Errorf("listing connectwise boards: %w", err)
	}
	slog.Debug("board sync: got boards from connectwise", "total_boards", len(cwb))

	sb, err := s.Boards.List(ctx)
	if err != nil {
		return fmt.Errorf("listing boards from store: %w", err)
	}
	slog.Debug("board sync: got boards from store", "total_boards", len(sb))

	//TODO: this is a bandaid. Move this logic to the repo.
	txSvc := s
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning tx: %w", err)
		}

		txSvc = s.withTX(tx)

		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()
	}

	var syncErrs []error
	for _, b := range s.boardsToUpsert(cwb) {
		if _, err := txSvc.Boards.Upsert(ctx, b); err != nil {
			syncErrs = append(syncErrs, fmt.Errorf("upserting board %d (%s): %w", b.ID, b.Name, err))
		}
	}

	for _, b := range s.boardsToDelete(cwb, sb) {
		if err := txSvc.Boards.Delete(ctx, b.ID); err != nil {
			syncErrs = append(syncErrs, fmt.Errorf("deleting board %d (%s): %w", b.ID, b.Name, err))
		}
	}

	if len(syncErrs) > 0 {
		for _, e := range syncErrs {
			slog.Error("board sync", "error", e)
		}
	}

	//TODO: this is a bandaid. Move this logic to the repo.
	if s.pool != nil {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing tx: %w", err)
		}
	}

	slog.Debug("board sync complete", "took_time", time.Since(start))
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
