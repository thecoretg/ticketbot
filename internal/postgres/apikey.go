package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type APIKeyRepo struct {
	queries *db.Queries
}

func NewAPIKeyRepo(pool *pgxpool.Pool) *APIKeyRepo {
	return &APIKeyRepo{queries: db.New(pool)}
}

func (p *APIKeyRepo) WithTx(tx pgx.Tx) models.APIKeyRepository {
	return &APIKeyRepo{queries: db.New(tx)}
}

func (p *APIKeyRepo) List(ctx context.Context) ([]models.APIKey, error) {
	dk, err := p.queries.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	var k []models.APIKey
	for _, d := range dk {
		k = append(k, keyFromPG(d))
	}

	return k, nil
}

func (p *APIKeyRepo) Get(ctx context.Context, id int) (models.APIKey, error) {
	d, err := p.queries.GetAPILKey(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.APIKey{}, models.ErrAPIKeyNotFound
		}
		return models.APIKey{}, models.ErrAPIKeyNotFound
	}

	return keyFromPG(d), nil
}

func (p *APIKeyRepo) Insert(ctx context.Context, a models.APIKey) (models.APIKey, error) {
	d, err := p.queries.InsertAPIKey(ctx, insertParamsFromAPIKey(a))
	if err != nil {
		return models.APIKey{}, err
	}

	return keyFromPG(d), nil
}

func (p *APIKeyRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteAPIKey(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAPIKeyNotFound
		}
		return err
	}

	return nil
}

func insertParamsFromAPIKey(a models.APIKey) db.InsertAPIKeyParams {
	return db.InsertAPIKeyParams{
		UserID:  a.UserID,
		KeyHash: a.KeyHash,
	}
}

func keyFromPG(pg db.ApiKey) models.APIKey {
	return models.APIKey{
		ID:        pg.ID,
		UserID:    pg.UserID,
		KeyHash:   pg.KeyHash,
		CreatedOn: pg.CreatedOn,
		UpdatedOn: pg.UpdatedOn,
	}
}
