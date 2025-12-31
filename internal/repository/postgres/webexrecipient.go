package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type WebexRecipientRepo struct {
	queries *db.Queries
}

func NewWebexRecipientRepo(pool *pgxpool.Pool) *WebexRecipientRepo {
	return &WebexRecipientRepo{
		queries: db.New(pool),
	}
}

func (p *WebexRecipientRepo) WithTx(tx pgx.Tx) models.WebexRecipientRepository {
	return &WebexRecipientRepo{
		queries: db.New(tx),
	}
}

func (p *WebexRecipientRepo) List(ctx context.Context) ([]*models.WebexRecipient, error) {
	dbr, err := p.queries.ListWebexRecipients(ctx)
	if err != nil {
		return nil, err
	}

	var r []*models.WebexRecipient
	for _, d := range dbr {
		r = append(r, recipFromPG(d))
	}

	return r, nil
}

func (p *WebexRecipientRepo) ListRooms(ctx context.Context) ([]*models.WebexRecipient, error) {
	dbr, err := p.queries.ListWebexRooms(ctx)
	if err != nil {
		return nil, err
	}

	var r []*models.WebexRecipient
	for _, d := range dbr {
		r = append(r, recipFromPG(d))
	}

	return r, nil
}

func (p *WebexRecipientRepo) ListPeople(ctx context.Context) ([]*models.WebexRecipient, error) {
	dbr, err := p.queries.ListWebexPeople(ctx)
	if err != nil {
		return nil, err
	}

	var r []*models.WebexRecipient
	for _, d := range dbr {
		r = append(r, recipFromPG(d))
	}

	return r, nil
}

func (p *WebexRecipientRepo) ListByEmail(ctx context.Context, email string) ([]*models.WebexRecipient, error) {
	dbr, err := p.queries.ListByEmail(ctx, &email)
	if err != nil {
		return nil, err
	}

	var r []*models.WebexRecipient
	for _, d := range dbr {
		r = append(r, recipFromPG(d))
	}

	return r, nil
}

func (p *WebexRecipientRepo) Get(ctx context.Context, id int) (*models.WebexRecipient, error) {
	d, err := p.queries.GetWebexRecipient(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWebexRecipientNotFound
		}
		return nil, err
	}

	return recipFromPG(d), nil
}

func (p *WebexRecipientRepo) GetByWebexID(ctx context.Context, webexID string) (*models.WebexRecipient, error) {
	d, err := p.queries.GetWebexRecipientByWebexID(ctx, webexID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWebexRecipientNotFound
		}
		return nil, err
	}

	return recipFromPG(d), nil
}

func (p *WebexRecipientRepo) Upsert(ctx context.Context, r *models.WebexRecipient) (*models.WebexRecipient, error) {
	d, err := p.queries.UpsertWebexRecipient(ctx, webexRoomToUpsertParams(r))
	if err != nil {
		return nil, err
	}

	return recipFromPG(d), nil
}

func (p *WebexRecipientRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteWebexRecipient(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrWebexRecipientNotFound
		}
		return err
	}

	return nil
}

func webexRoomToUpsertParams(r *models.WebexRecipient) db.UpsertWebexRecipientParams {
	return db.UpsertWebexRecipientParams{
		WebexID:      r.WebexID,
		Name:         r.Name,
		Type:         string(r.Type),
		Email:        r.Email,
		LastActivity: r.LastActivity,
	}
}

func recipFromPG(pg *db.WebexRecipient) *models.WebexRecipient {
	return &models.WebexRecipient{
		ID:           pg.ID,
		WebexID:      pg.WebexID,
		Name:         pg.Name,
		Type:         models.WebexRecipientType(pg.Type),
		Email:        pg.Email,
		LastActivity: pg.LastActivity,
		CreatedOn:    pg.CreatedOn,
		UpdatedOn:    pg.UpdatedOn,
	}
}
