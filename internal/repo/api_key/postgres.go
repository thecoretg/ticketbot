package api_key

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
)

type PostgresRepo struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (p *PostgresRepo) List(ctx context.Context) ([]APIKey, error) {
	dk, err := p.queries.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	var k []APIKey
	for _, d := range dk {
		k = append(k, keyFromPG(d))
	}

	return k, nil
}

func (p *PostgresRepo) Get(ctx context.Context, id int) (APIKey, error) {
	d, err := p.queries.GetAPILKey(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return APIKey{}, ErrNotFound
		}
		return APIKey{}, ErrNotFound
	}

	return keyFromPG(d), nil
}

func (p *PostgresRepo) Insert(ctx context.Context, a APIKey) (APIKey, error) {
	d, err := p.queries.InsertAPIKey(ctx, pgInsertParams(a))
	if err != nil {
		return APIKey{}, err
	}

	return keyFromPG(d), nil
}

func (p *PostgresRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteAPIKey(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func pgInsertParams(a APIKey) db.InsertAPIKeyParams {
	return db.InsertAPIKeyParams{
		UserID:  a.UserID,
		KeyHash: a.KeyHash,
	}
}

func keyFromPG(pg db.ApiKey) APIKey {
	return APIKey{
		ID:        pg.ID,
		UserID:    pg.UserID,
		KeyHash:   pg.KeyHash,
		CreatedOn: pg.CreatedOn,
		UpdatedOn: pg.UpdatedOn,
	}
}
