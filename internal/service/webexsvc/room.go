package webexsvc

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

func (s *Service) ListRooms(ctx context.Context) ([]models.WebexRecipient, error) {
	return s.Rooms.List(ctx)
}

func (s *Service) GetRoom(ctx context.Context, id int) (models.WebexRecipient, error) {
	return s.Rooms.Get(ctx, id)
}

func (s *Service) SyncRooms(ctx context.Context) error {
	// TODO: this function is gross and needs to be split up in some way.

	start := time.Now()
	slog.Info("beginning webex room sync")

	// get rooms from webex as source of truth
	wr, err := s.webexClient.ListRooms(nil)
	if err != nil {
		return fmt.Errorf("getting rooms from webex: %w", err)
	}
	slog.Info("webex room sync: got rooms from webex", "total_rooms", len(wr))

	// get current rooms from store
	sr, err := s.Rooms.List(ctx)
	if err != nil {
		return fmt.Errorf("getting rooms from store: %w", err)
	}
	slog.Info("webex room sync: got rooms from store", "total_rooms", len(sr))

	// TODO: this is a bandaid. Move this logic to the repo.
	txSvc := s
	var tx pgx.Tx
	if s.pool != nil {
		tx, err = s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("beginning tx: %w", err)
		}

		txSvc = s.withTx(tx)

		defer func() {
			_ = tx.Rollback(ctx)
		}()
	}

	for _, r := range roomsToUpsert(wr) {
		if _, err := txSvc.Rooms.Upsert(ctx, r); err != nil {
			return fmt.Errorf("upserting room with name %s: %w", r.Name, err)
		}
	}

	// TODO: this is a bandaid. Move this logic to the repo.
	if s.pool != nil {
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("committing tx: %w", err)
		}
	}

	slog.Info("webex room sync complete", "took_time", time.Since(start).Seconds())
	return nil
}

func roomsToUpsert(webexRooms []webex.Room) []models.WebexRecipient {
	var toUpsert []models.WebexRecipient
	for _, w := range webexRooms {
		r := models.WebexRecipient{
			WebexID:      w.Id,
			Name:         w.Title,
			Type:         w.Type,
			LastActivity: w.LastActivity,
		}
		toUpsert = append(toUpsert, r)
	}

	return toUpsert
}
