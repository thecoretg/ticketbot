package server

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/repository/inmem"
	"github.com/thecoretg/ticketbot/internal/repository/postgres"
	"github.com/thecoretg/ticketbot/migrations"
)

type Stores struct {
	Repos *models.AllRepos
	Pool  *pgxpool.Pool
}

func CreateStores(ctx context.Context, creds *Creds, inMemory bool) (*Stores, error) {
	if inMemory {
		slog.Info("running with in-memory store")
		return InitInMemStores(), nil
	}

	slog.Info("running with postgres store")
	return InitPostgresStores(ctx, creds)
}

// InitPostgresStores verifies credentials are given, runs any needed migrations, and
// provides all repositories
func InitPostgresStores(ctx context.Context, creds *Creds) (*Stores, error) {
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

	return &Stores{
		Pool:  pool,
		Repos: postgres.AllRepos(pool),
	}, nil
}

func InitInMemStores() *Stores {
	return &Stores{
		Repos: inmem.AllRepos(nil),
		Pool:  nil,
	}
}
