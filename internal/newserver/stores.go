package newserver

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/repository/inmem"
	"github.com/thecoretg/ticketbot/internal/repository/postgres"
	"github.com/thecoretg/ticketbot/migrations"
)

type storesResult struct {
	stores *models.AllRepos
	pool   *pgxpool.Pool
}

func initStores(ctx context.Context, creds *creds, inMemory bool) (*storesResult, error) {
	if inMemory {
		return initInMem(), nil
	}

	return initPostgres(ctx, creds)
}

// initPostgres verifies credentials are given, runs any needed migrations, and
// provides all repositories
func initPostgres(ctx context.Context, creds *creds) (*storesResult, error) {
	pool, err := pgxpool.New(ctx, creds.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}

	m, err := fs.Sub(migrations.Migrations, ".")
	if err != nil {
		return nil, fmt.Errorf("connecting/migrating db: %w", err)
	}

	d := stdlib.OpenDBFromPool(pool)

	goose.SetBaseFS(m)
	if err := goose.SetDialect("postgres"); err != nil {
		return nil, fmt.Errorf("setting goose dialect: %w", err)
	}

	if err := goose.Up(d, "."); err != nil {
		return nil, fmt.Errorf("running goose-up: %w", err)
	}

	return &storesResult{
		pool:   pool,
		stores: postgres.AllRepos(pool),
	}, nil
}

func initInMem() *storesResult {
	return &storesResult{
		stores: inmem.AllRepos(nil),
		pool:   nil,
	}
}
