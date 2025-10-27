package server

import (
	"database/sql"
	"embed"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

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
