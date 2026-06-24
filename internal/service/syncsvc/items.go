package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/tctg-go/connectwise/psa"
)

func (s *Service) SyncBoardItems(ctx context.Context, boardID int) error {
	start := time.Now()
	slog.Info("beginning connectwise item sync", "board_id", boardID)
	cwi, err := s.CW.CWClient.ListBoardItems(ctx, nil, boardID)
	if err != nil {
		return fmt.Errorf("listing connectwise items for board %d: %w", boardID, err)
	}
	slog.Info("item sync: got items from connectwise", "board_id", boardID, "total_items", len(cwi))

	si, err := s.CW.Items.ListByBoard(ctx, boardID)
	if err != nil {
		return fmt.Errorf("listing items from store: %w", err)
	}
	slog.Info("item sync: got items from store", "board_id", boardID, "total_items", len(si))

	for _, i := range itemsToUpsert(cwi) {
		if _, err := s.CW.Items.Upsert(ctx, i); err != nil {
			return fmt.Errorf("upserting item %d (%s): %w", i.ID, i.Name, err)
		}
	}

	for _, i := range itemsToDelete(cwi, si) {
		if err := s.CW.Items.SoftDelete(ctx, i.ID); err != nil {
			return fmt.Errorf("soft deleting item %d (%s): %w", i.ID, i.Name, err)
		}
	}

	slog.Info("item sync: complete", "board_id", boardID, "took_time", time.Since(start).Seconds())
	return nil
}

func itemsToUpsert(cwItems []psa.BoardItem) []*models.TicketItem {
	var toUpsert []*models.TicketItem
	for _, c := range cwItems {
		i := &models.TicketItem{
			ID:       c.ID,
			BoardID:  c.Board.ID,
			Name:     c.Name,
			Inactive: c.InactiveFlag,
		}
		toUpsert = append(toUpsert, i)
	}

	return toUpsert
}

func itemsToDelete(cwItems []psa.BoardItem, storeItems []*models.TicketItem) []*models.TicketItem {
	ci := make(map[int]psa.BoardItem)
	for _, c := range cwItems {
		ci[c.ID] = c
	}

	var toDelete []*models.TicketItem
	for _, i := range storeItems {
		// skip soft deleted items
		if i.Deleted {
			continue
		}

		if _, ok := ci[i.ID]; !ok {
			toDelete = append(toDelete, i)
		}
	}

	return toDelete
}
