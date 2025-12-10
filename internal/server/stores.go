package server

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/repository/postgres"
	"github.com/thecoretg/ticketbot/migrations"
)

type Stores struct {
	Repos *models.AllRepos
	Pool  *pgxpool.Pool
}

func CreateStores(ctx context.Context, creds *Creds, targetMigVersion int64) (*Stores, error) {
	slog.Info("running with postgres store")
	return InitPostgresStores(ctx, creds, targetMigVersion)
}

// InitPostgresStores verifies credentials are given, runs any needed migrations, and
// provides all repositories
func InitPostgresStores(ctx context.Context, creds *Creds, targetMigVersion int64) (*Stores, error) {
	pool, err := pgxpool.New(ctx, creds.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}

	d := stdlib.OpenDBFromPool(pool)
	if err := GooseMigrate(d, targetMigVersion); err != nil {
		return nil, fmt.Errorf("migrating database: %w", err)
	}

	return &Stores{
		Pool:  pool,
		Repos: postgres.AllRepos(pool),
	}, nil
}

func GooseMigrate(d *sql.DB, target int64) error {
	m, err := fs.Sub(migrations.Migrations, ".")
	if err != nil {
		return fmt.Errorf("connecting/migrating db: %w", err)
	}

	goose.SetBaseFS(m)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	current, err := goose.EnsureDBVersion(d)
	if err != nil {
		return fmt.Errorf("goose: checking db version: %w", err)
	}

	if current > target {
		slog.Info("goose: migrating down", "target", target, "current", current)
		err = goose.DownTo(d, ".", target)
	} else if current < target {
		slog.Info("goose: migrating up", "target", target, "current", current)
		err = goose.UpTo(d, ".", target)
	} else {
		slog.Info("target and current db versions match", "target", target, "current", current)
	}

	return err
}
