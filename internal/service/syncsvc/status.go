package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func (s *Service) SyncBoardStatuses(ctx context.Context, boardID int) error {
	start := time.Now()
	slog.Info("beginning connectwise status sync", "board_id", boardID)
	cws, err := s.CW.CWClient.ListBoardStatuss(nil, boardID)
	if err != nil {
		return fmt.Errorf("listing connectwise statuses for board %d: %w", boardID, err)
	}
	slog.Info("status sync: got statuses from connectwise", "board_id", boardID, "total_boards", len(cws))

	sst, err := s.CW.Statuses.ListByBoard(ctx, boardID)
	if err != nil {
		return fmt.Errorf("listing statuses from store: %w", err)
	}
	slog.Info("status sync: got statuses from store", "board_id", boardID, "total_boards", len(sst))

	for _, b := range statusesToUpsert(cws) {
		if _, err := s.CW.Statuses.Upsert(ctx, b); err != nil {
			return fmt.Errorf("upserting status %d (%s): %w", b.ID, b.Name, err)
		}
	}

	for _, b := range statusesToDelete(cws, sst) {
		if err := s.CW.Statuses.SoftDelete(ctx, b.ID); err != nil {
			return fmt.Errorf("soft deleting status %d (%s): %w", b.ID, b.Name, err)
		}
	}

	slog.Info("status sync: complete", "board_id", boardID, "took_time", time.Since(start).Seconds())
	return nil
}

func statusesToUpsert(cwStatuses []psa.BoardStatus) []*models.TicketStatus {
	var toUpsert []*models.TicketStatus
	for _, c := range cwStatuses {
		b := &models.TicketStatus{
			ID:             c.ID,
			BoardID:        c.Board.ID,
			Name:           c.Name,
			DefaultStatus:  c.DefaultFlag,
			DisplayOnBoard: c.DisplayOnBoard,
			Inactive:       c.Inactive,
			Closed:         c.ClosedStatus,
		}
		toUpsert = append(toUpsert, b)
	}

	return toUpsert
}

func statusesToDelete(cwStatuses []psa.BoardStatus, storeStatuses []*models.TicketStatus) []*models.TicketStatus {
	ci := make(map[int]psa.BoardStatus)
	for _, c := range cwStatuses {
		ci[c.ID] = c
	}

	var toDelete []*models.TicketStatus
	for _, s := range storeStatuses {
		// skip soft deleted statuses
		if s.Deleted {
			continue
		}

		if _, ok := ci[s.ID]; !ok {
			toDelete = append(toDelete, s)
		}
	}

	return toDelete
}
