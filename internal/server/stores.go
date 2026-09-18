package server

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/postgres"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/migrations"
)

type Stores struct {
	Repos *repos.AllRepos
	Pool  *pgxpool.Pool
}

// InitPostgresStores opens the connection pool, migrates the schema to targetMigVersion, and
// provides all repositories.
func InitPostgresStores(ctx context.Context, e *env.Env, targetMigVersion int64) (*Stores, error) {
	pc, err := pgxpool.ParseConfig(e.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("parsing postgres dsn: %w", err)
	}
	pc.MaxConns = e.PostgresMaxConns
	pc.MinConns = min(2, e.PostgresMaxConns)
	pc.MaxConnLifetime = time.Hour
	pc.MaxConnIdleTime = 5 * time.Minute
	pc.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, pc)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connecting to postgres: %w", err)
	}

	d := stdlib.OpenDBFromPool(pool)
	if err := GooseMigrate(d, targetMigVersion); err != nil {
		pool.Close()
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
