package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/tctg-go/connectwise/psa"
)

func (s *Service) SyncBoardSubTypes(ctx context.Context, boardID int) error {
	start := time.Now()
	slog.Info("beginning connectwise subtype sync", "board_id", boardID)
	cwst, err := s.CW.CWClient.ListBoardSubTypes(ctx, nil, boardID)
	if err != nil {
		return fmt.Errorf("listing connectwise subtypes for board %d: %w", boardID, err)
	}
	slog.Info("subtype sync: got subtypes from connectwise", "board_id", boardID, "total_subtypes", len(cwst))

	sst, err := s.CW.SubTypes.ListByBoard(ctx, boardID)
	if err != nil {
		return fmt.Errorf("listing subtypes from store: %w", err)
	}
	slog.Info("subtype sync: got subtypes from store", "board_id", boardID, "total_subtypes", len(sst))

	for _, st := range subTypesToUpsert(cwst) {
		if _, err := s.CW.SubTypes.Upsert(ctx, st); err != nil {
			return fmt.Errorf("upserting subtype %d (%s): %w", st.ID, st.Name, err)
		}
	}

	for _, st := range subTypesToDelete(cwst, sst) {
		if err := s.CW.SubTypes.SoftDelete(ctx, st.ID); err != nil {
			return fmt.Errorf("soft deleting subtype %d (%s): %w", st.ID, st.Name, err)
		}
	}

	slog.Info("subtype sync: complete", "board_id", boardID, "took_time", time.Since(start).Seconds())
	return nil
}

func subTypesToUpsert(cwSubTypes []psa.BoardSubType) []*models.TicketSubType {
	var toUpsert []*models.TicketSubType
	for _, c := range cwSubTypes {
		st := &models.TicketSubType{
			ID:                 c.ID,
			BoardID:            c.Board.ID,
			Name:               c.Name,
			Inactive:           c.InactiveFlag,
			TypeAssociationIDs: c.TypeAssociationIds,
		}
		toUpsert = append(toUpsert, st)
	}

	return toUpsert
}

func subTypesToDelete(cwSubTypes []psa.BoardSubType, storeSubTypes []*models.TicketSubType) []*models.TicketSubType {
	ci := make(map[int]psa.BoardSubType)
	for _, c := range cwSubTypes {
		ci[c.ID] = c
	}

	var toDelete []*models.TicketSubType
	for _, st := range storeSubTypes {
		// skip soft deleted subtypes
		if st.Deleted {
			continue
		}

		if _, ok := ci[st.ID]; !ok {
			toDelete = append(toDelete, st)
		}
	}

	return toDelete
}
