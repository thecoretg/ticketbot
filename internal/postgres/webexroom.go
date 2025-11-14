package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type WebexRoomRepo struct {
	queries *db.Queries
}

func NewWebexRoomRepo(pool *pgxpool.Pool) *WebexRoomRepo {
	return &WebexRoomRepo{
		queries: db.New(pool),
	}
}

func (p *WebexRoomRepo) WithTx(tx pgx.Tx) models.WebexRoomRepository {
	return &WebexRoomRepo{
		queries: db.New(tx)}
}

func (p *WebexRoomRepo) List(ctx context.Context) ([]models.WebexRoom, error) {
	dbr, err := p.queries.ListWebexRooms(ctx)
	if err != nil {
		return nil, err
	}

	var r []models.WebexRoom
	for _, d := range dbr {
		r = append(r, roomFromPG(d))
	}

	return r, nil
}

func (p *WebexRoomRepo) Get(ctx context.Context, id int) (models.WebexRoom, error) {
	d, err := p.queries.GetWebexRoom(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WebexRoom{}, models.ErrWebexRoomNotFound
		}
		return models.WebexRoom{}, err
	}

	return roomFromPG(d), nil
}

func (p *WebexRoomRepo) GetByWebexID(ctx context.Context, webexID string) (models.WebexRoom, error) {
	d, err := p.queries.GetWebexRoomByWebexID(ctx, webexID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.WebexRoom{}, models.ErrWebexRoomNotFound
		}
		return models.WebexRoom{}, err
	}

	return roomFromPG(d), nil
}

func (p *WebexRoomRepo) Upsert(ctx context.Context, r models.WebexRoom) (models.WebexRoom, error) {
	d, err := p.queries.UpsertWebexRoom(ctx, webexRoomToUpsertParams(r))
	if err != nil {
		return models.WebexRoom{}, err
	}

	return roomFromPG(d), nil
}

func (p *WebexRoomRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteWebexRoom(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrWebexRoomNotFound
		}
		return err
	}

	return nil
}

func webexRoomToUpsertParams(r models.WebexRoom) db.UpsertWebexRoomParams {
	return db.UpsertWebexRoomParams{
		WebexID: r.WebexID,
		Name:    r.Name,
		Type:    r.Type,
	}
}

func roomFromPG(pg db.WebexRoom) models.WebexRoom {
	return models.WebexRoom{
		ID:        pg.ID,
		WebexID:   pg.WebexID,
		Name:      pg.Name,
		Type:      pg.Type,
		CreatedOn: pg.CreatedOn,
		UpdatedOn: pg.UpdatedOn,
	}
}
