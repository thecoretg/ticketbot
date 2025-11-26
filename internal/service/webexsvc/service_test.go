package webexsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/thecoretg/ticketbot/internal/repository/postgres"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

func TestNewService(t *testing.T) {
	if _, err := newTestService(t, context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestService_SyncRooms(t *testing.T) {
	ctx := context.Background()
	s, err := newTestService(t, ctx)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	if err := s.SyncRooms(ctx); err != nil {
		t.Fatalf("syncing rooms: %v", err)
	}
}

func newTestService(t *testing.T, ctx context.Context) (*Service, error) {
	t.Helper()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	if err := godotenv.Load("testing.env"); err != nil {
		return nil, fmt.Errorf("loading .env")
	}

	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, errors.New("postgres dsn is empty")
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("creating pgx pool: %w", err)
	}

	t.Cleanup(func() { pool.Close() })

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging pool")
	}

	secret := os.Getenv("WEBEX_SECRET")

	return New(pool, postgres.NewWebexRoomRepo(pool), webex.NewClient(secret)), nil
}
