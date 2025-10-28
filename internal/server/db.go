package server

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// setupDB configures the pgx pool and runs any needed data migrations via goose.
// It returns the pgx pool, which will be needed to configure the Client struct.
func setupDB(ctx context.Context, dsn string, embeddedMigrations embed.FS) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}

	if err := migrateDB(pool, embeddedMigrations); err != nil {
		return nil, fmt.Errorf("connecting/migrating db: %w", err)
	}

	return pool, nil
}

func migrateDB(pool *pgxpool.Pool, embeddedMigrations embed.FS) error {
	goose.SetBaseFS(embeddedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("setting goose dialect: %w", err)
	}

	d := stdlib.OpenDBFromPool(pool)
	defer func(d *sql.DB) {
		err := d.Close()
		if err != nil {
			slog.Error("error closing db", "error", err)
		}
	}(d)

	if err := goose.Up(d, "migrations"); err != nil {
		return fmt.Errorf("running goose up: %w", err)
	}

	return nil
}
