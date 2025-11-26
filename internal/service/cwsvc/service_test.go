package cwsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/repository/inmem"
	"github.com/thecoretg/ticketbot/internal/repository/postgres"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func TestNewService(t *testing.T) {
	if _, err := newInmemTestService(t, context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestService_SyncBoards(t *testing.T) {
	ctx := context.Background()
	s, err := newInmemTestService(t, ctx)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	if err := s.SyncBoards(ctx); err != nil {
		t.Fatalf("syncing boards: %v", err)
	}

	boards, err := s.Boards.List(ctx)
	if err != nil {
		t.Fatalf("listing boards (second pass): %v", err)
	}

	t.Logf("got %d boards (second pass)", len(boards))
}

func TestService_ProcessTicket(t *testing.T) {
	ctx := context.Background()
	s, err := newInmemTestService(t, ctx)
	if err != nil {
		t.Fatalf("creating service: %v", err)
	}

	ids := testTicketIDs(t)
	// add
	for _, id := range ids {
		if _, err := s.ProcessTicket(ctx, id); err != nil {
			t.Errorf("adding ticket %d: %v", id, err)
			continue
		}
	}

	// update (not changing anything, just simualting)
	for _, id := range ids {
		if _, err := s.ProcessTicket(ctx, id); err != nil {
			t.Errorf("updating ticket %d: %v", id, err)
			continue
		}
	}

	// delete
	for _, id := range ids {
		if err := s.DeleteTicket(ctx, id); err != nil {
			t.Errorf("deleting ticket %d: %v", id, err)
			continue
		}
	}
}

func newInmemTestService(t *testing.T, ctx context.Context) (*Service, error) {
	t.Helper()
	if err := godotenv.Load("testing.env"); err != nil {
		return nil, fmt.Errorf("loading .env")
	}

	r := models.CWRepos{
		Board:   inmem.NewBoardRepo(nil),
		Company: inmem.NewCompanyRepo(nil),
		Contact: inmem.NewContactRepo(nil),
		Member:  inmem.NewMemberRepo(nil),
		Note:    inmem.NewTicketNoteRepo(nil),
		Ticket:  inmem.NewTicketRepo(nil),
	}

	return New(nil, r, psa.NewClient(testGetCwCreds(t))), nil
}

func newPostgresTestService(t *testing.T, ctx context.Context) (*Service, error) {
	t.Helper()
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

	r := models.CWRepos{
		Board:   postgres.NewBoardRepo(pool),
		Company: postgres.NewCompanyRepo(pool),
		Contact: postgres.NewContactRepo(pool),
		Member:  postgres.NewMemberRepo(pool),
		Note:    postgres.NewTicketNoteRepo(pool),
		Ticket:  postgres.NewTicketRepo(pool),
	}

	return New(pool, r, psa.NewClient(testGetCwCreds(t))), nil
}

func testGetCwCreds(t *testing.T) *psa.Creds {
	t.Helper()
	return &psa.Creds{
		PublicKey:  os.Getenv("CW_PUB_KEY"),
		PrivateKey: os.Getenv("CW_PRIV_KEY"),
		ClientId:   os.Getenv("CW_CLIENT_ID"),
		CompanyId:  os.Getenv("CW_COMPANY_ID"),
	}
}

func testTicketIDs(t *testing.T) []int {
	t.Helper()
	slog.SetLogLoggerLevel(slog.LevelDebug)
	raw := os.Getenv("TEST_TICKET_IDS")
	split := strings.Split(raw, ",")

	var ids []int
	for _, s := range split {
		i, err := strconv.Atoi(s)
		if err != nil {
			t.Logf("couldn't convert ticket id '%s' to integer", s)
			continue
		}
		ids = append(ids, i)
	}

	return ids
}
