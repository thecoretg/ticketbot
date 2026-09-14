package postgres

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/models"
)

// testPool connects to TEST_POSTGRES_DSN, which must point at a database already migrated to the
// current version. Tests are skipped when it is unset.
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN not set")
	}

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	return pool
}

func seedTicket(t *testing.T, pool *pgxpool.Pool, id int, summary string) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO cw_board (id, name) VALUES (900001, 'Help Desk') ON CONFLICT DO NOTHING;
		INSERT INTO cw_ticket_status (id, board_id, name, default_status, display_on_board, inactive, closed)
			VALUES (900010, 900001, 'New', true, true, false, false) ON CONFLICT DO NOTHING;
		INSERT INTO cw_company (id, name) VALUES (900100, 'Acme') ON CONFLICT DO NOTHING;
		INSERT INTO cw_member (id, identifier, first_name, last_name, primary_email)
			VALUES (900005, 'jdoe', 'Jane', 'Doe', 'jane@example.com') ON CONFLICT DO NOTHING;`)
	if err != nil {
		t.Fatal(err)
	}

	owner := 900005
	rsc := "jdoe"
	raw, _ := json.Marshal(map[string]any{"id": id, "summary": summary})
	_, err = NewTicketRepo(pool).Upsert(ctx, &models.Ticket{
		ID: id, Summary: summary, BoardID: 900001, StatusID: 900010, OwnerID: &owner, CompanyID: 900100,
		Resources: &rsc, Raw: raw,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM cw_ticket WHERE id = $1`, id) })
}

func TestTicketRepoListPaged(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	seedTicket(t, pool, 900001, "Integration printer jam")
	seedTicket(t, pool, 900002, "Integration VPN outage")

	repo := NewTicketRepo(pool)

	items, total, err := repo.ListPaged(ctx, models.TicketFilter{Search: "Integration", PageSize: 1, Page: 1})
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(items) != 1 {
		t.Fatalf("total=%d len=%d, want 2/1", total, len(items))
	}
	if items[0].BoardName != "Help Desk" || items[0].StatusName != "New" || items[0].CompanyName != "Acme" {
		t.Errorf("names not joined: %+v", items[0])
	}
	if items[0].OwnerName != "Jane Doe" {
		t.Errorf("owner name = %q", items[0].OwnerName)
	}

	items, total, err = repo.ListPaged(ctx, models.TicketFilter{Search: "900002"})
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || items[0].ID != 900002 {
		t.Errorf("id search: total=%d items=%+v", total, items)
	}

	got, err := repo.Get(ctx, 900001)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Raw) == 0 {
		t.Error("raw not stored")
	}
}

func TestTicketEventRepo(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	seedTicket(t, pool, 900003, "Integration events")

	repo := NewTicketEventRepo(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	var events []*models.TicketEvent
	for i, k := range []models.EventKind{models.EventCreated, models.EventUpdated, models.EventAction} {
		e, err := models.NewTicketEvent(900003, "run-1", k, models.SourceWebhook, i == 2,
			models.ActionPayload{RuleName: "r", Result: "ok"}, now.Add(time.Duration(i)*time.Second))
		if err != nil {
			t.Fatal(err)
		}
		events = append(events, e)
	}
	if err := repo.InsertBatch(ctx, events); err != nil {
		t.Fatal(err)
	}

	got, err := repo.ListByTicket(ctx, 900003, 10, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("got %d events", len(got))
	}
	if got[0].Kind != models.EventAction || !got[0].DryRun {
		t.Errorf("newest first expected action/dry_run, got %+v", got[0])
	}
	var p models.ActionPayload
	if err := json.Unmarshal(got[0].Payload, &p); err != nil || p.RuleName != "r" {
		t.Errorf("payload round trip failed: %v %+v", err, p)
	}

	page, err := repo.ListByTicket(ctx, 900003, 10, &got[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(page) != 2 {
		t.Errorf("before_id paging returned %d", len(page))
	}

	single, err := repo.Insert(ctx, &models.TicketEvent{TicketID: 900003, RunID: "run-2", Kind: models.EventDeleted, Source: models.SourceSync, OccurredAt: now})
	if err != nil {
		t.Fatal(err)
	}
	if single.ID == 0 || string(single.Payload) != "{}" {
		t.Errorf("single insert: %+v", single)
	}
}
