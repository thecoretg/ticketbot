package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/tctg-go/connectwise/psa"
)

func (s *Service) SyncBoardTypes(ctx context.Context, boardID int) error {
	start := time.Now()
	slog.Info("beginning connectwise type sync", "board_id", boardID)
	cwt, err := s.CW.CWClient.ListBoardTypes(ctx, nil, boardID)
	if err != nil {
		return fmt.Errorf("listing connectwise types for board %d: %w", boardID, err)
	}
	slog.Info("type sync: got types from connectwise", "board_id", boardID, "total_types", len(cwt))

	st, err := s.CW.Types.ListByBoard(ctx, boardID)
	if err != nil {
		return fmt.Errorf("listing types from store: %w", err)
	}
	slog.Info("type sync: got types from store", "board_id", boardID, "total_types", len(st))

	for _, t := range typesToUpsert(cwt) {
		if _, err := s.CW.Types.Upsert(ctx, t); err != nil {
			return fmt.Errorf("upserting type %d (%s): %w", t.ID, t.Name, err)
		}
	}

	for _, t := range typesToDelete(cwt, st) {
		if err := s.CW.Types.SoftDelete(ctx, t.ID); err != nil {
			return fmt.Errorf("soft deleting type %d (%s): %w", t.ID, t.Name, err)
		}
	}

	slog.Info("type sync: complete", "board_id", boardID, "took_time", time.Since(start).Seconds())
	return nil
}

func typesToUpsert(cwTypes []psa.BoardType) []*models.TicketType {
	var toUpsert []*models.TicketType
	for _, c := range cwTypes {
		t := &models.TicketType{
			ID:          c.ID,
			BoardID:     c.Board.ID,
			Name:        c.Name,
			Category:    c.Category,
			DefaultFlag: c.DefaultFlag,
			Inactive:    c.InactiveFlag,
		}
		toUpsert = append(toUpsert, t)
	}

	return toUpsert
}

func typesToDelete(cwTypes []psa.BoardType, storeTypes []*models.TicketType) []*models.TicketType {
	ci := make(map[int]psa.BoardType)
	for _, c := range cwTypes {
		ci[c.ID] = c
	}

	var toDelete []*models.TicketType
	for _, t := range storeTypes {
		// skip soft deleted types
		if t.Deleted {
			continue
		}

		if _, ok := ci[t.ID]; !ok {
			toDelete = append(toDelete, t)
		}
	}

	return toDelete
}
