package webexsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
)

func (s *Service) ListRooms(ctx context.Context) ([]models.WebexRoom, error) {
	return s.Rooms.List(ctx)
}

func (s *Service) GetRoom(ctx context.Context, id int) (models.WebexRoom, error) {
	return s.Rooms.Get(ctx, id)
}

func (s *Service) GetRoomByWebexID(ctx context.Context, id string) (models.WebexRoom, error) {
	return s.Rooms.GetByWebexID(ctx, id)
}

func (s *Service) UpsertRoom(ctx context.Context, r models.WebexRoom) (models.WebexRoom, error) {
	return s.Rooms.Upsert(ctx, r)
}

func (s *Service) DeleteRoom(ctx context.Context, id int) error {
	return s.Rooms.Delete(ctx, id)
}

func (s *Service) SyncRooms(ctx context.Context) error {
	start := time.Now()
	slog.Debug("beginning webex room sync")

	// get rooms from webex as source of truth
	wr, err := s.webexClient.ListRooms(nil)
	if err != nil {
		return fmt.Errorf("getting rooms from webex: %w", err)
	}
	slog.Debug("webex room sync: got rooms from webex", "total_rooms", len(wr))

	// get current rooms from store
	sr, err := s.Rooms.List(ctx)
	if err != nil {
		return fmt.Errorf("getting rooms from store: %w", err)
	}
	slog.Debug("webex room sync: got rooms from store", "total_rooms", len(sr))

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning tx: %w", err)
	}

	txSvc := s.withTx(tx)

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var updateErrs []error
	for _, r := range roomsToUpsert(wr) {
		if _, err := txSvc.Rooms.Upsert(ctx, r); err != nil {
			updateErrs = append(updateErrs, fmt.Errorf("upserting room with name %s: %w", r.Name, err))
		}
	}

	var syncErr error
	if len(updateErrs) > 0 {
		for _, e := range updateErrs {
			slog.Error("webex room sync", "error", e)
		}
		syncErr = errors.New("error occured while syncing one or more rooms, see logs")
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	slog.Debug("webex room sync complete", "took_time", time.Since(start))
	return syncErr
}

func roomsToUpsert(webexRooms []webex.Room) []models.WebexRoom {
	var toUpsert []models.WebexRoom
	for _, w := range webexRooms {
		r := models.WebexRoom{
			WebexID: w.Id,
			Name:    w.Title,
			Type:    w.Type,
		}
		toUpsert = append(toUpsert, r)
	}

	return toUpsert
}
