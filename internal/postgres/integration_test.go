package postgres

import (
	"context"
	"encoding/json"
	"errors"
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
			models.ActionPayload{Title: "r", Result: "ok"}, now.Add(time.Duration(i)*time.Second))
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
	if err := json.Unmarshal(got[0].Payload, &p); err != nil || p.Title != "r" {
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

func TestCompanyAndContactSearch(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO cw_company (id, name) VALUES (900200, 'Zeta Widgets'), (900201, 'Zeta Gadgets') ON CONFLICT DO NOTHING;
		INSERT INTO cw_company (id, name, deleted) VALUES (900202, 'Zeta Gone', true) ON CONFLICT DO NOTHING;
		INSERT INTO cw_contact (id, first_name, last_name, company_id)
			VALUES (900300, 'Quinn', 'Zzyzxbrowski', 900200), (900301, 'Quill', 'Zzyzxother', 900201) ON CONFLICT DO NOTHING;`)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, `DELETE FROM cw_contact WHERE id IN (900300, 900301); DELETE FROM cw_company WHERE id IN (900200, 900201, 900202)`)
	})

	companies, err := NewCompanyRepo(pool).Search(ctx, models.CompanySearch{Query: "zeta"})
	if err != nil {
		t.Fatal(err)
	}
	if len(companies) != 2 || companies[0].Name != "Zeta Gadgets" || companies[1].Name != "Zeta Widgets" {
		t.Errorf("company search = %+v", companies)
	}

	byID, err := NewCompanyRepo(pool).Search(ctx, models.CompanySearch{IDs: []int{900201}})
	if err != nil || len(byID) != 1 || byID[0].ID != 900201 {
		t.Errorf("company by id = %+v, %v", byID, err)
	}

	contacts, err := NewContactRepo(pool).Search(ctx, models.ContactSearch{Query: "zzyzx"})
	if err != nil || len(contacts) != 2 {
		t.Fatalf("contact search = %+v, %v", contacts, err)
	}
	company := 900200
	scoped, err := NewContactRepo(pool).Search(ctx, models.ContactSearch{Query: "zzyzx", CompanyID: &company})
	if err != nil || len(scoped) != 1 || scoped[0].ID != 900300 {
		t.Errorf("scoped contact search = %+v, %v", scoped, err)
	}
	full, err := NewContactRepo(pool).Search(ctx, models.ContactSearch{Query: "quinn zzyzx"})
	if err != nil || len(full) != 1 {
		t.Errorf("full-name contact search = %+v, %v", full, err)
	}
}

func TestListRepo(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM list WHERE LOWER(name) LIKE 'it-test %'`) })
	_, _ = pool.Exec(ctx, `DELETE FROM list WHERE LOWER(name) LIKE 'it-test %'`)

	repo := NewListRepo(pool)
	l, err := repo.Insert(ctx, &models.List{Name: "IT-Test Drop Notifications", ItemType: models.ListItemContact, Description: "d"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Insert(ctx, &models.List{Name: "it-test drop notifications", ItemType: models.ListItemContact}); !errors.Is(err, models.ErrListNameTaken) {
		t.Errorf("duplicate insert err = %v, want ErrListNameTaken", err)
	}

	added, err := repo.AddItem(ctx, l.ID, 900300)
	if err != nil || !added {
		t.Fatalf("AddItem = %v, %v", added, err)
	}
	added, err = repo.AddItem(ctx, l.ID, 900300)
	if err != nil || added {
		t.Errorf("second AddItem = %v, %v; want false, nil", added, err)
	}
	if _, err := repo.AddItem(ctx, 0, 1); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("AddItem to unknown list err = %v", err)
	}

	items, err := repo.Items(ctx, l.ID)
	if err != nil || len(items) != 1 || items[0].ItemID != 900300 {
		t.Errorf("Items = %+v, %v", items, err)
	}
	got, err := repo.Get(ctx, l.ID)
	if err != nil || got.ItemCount != 1 || got.ItemType != models.ListItemContact {
		t.Errorf("Get = %+v, %v", got, err)
	}

	all, err := repo.AllMemberships(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range all {
		if m.ListID == l.ID && m.ItemID == 900300 {
			found = true
		}
	}
	if !found {
		t.Errorf("AllMemberships missing %d/900300: %+v", l.ID, all)
	}

	upd, err := repo.Update(ctx, &models.List{ID: l.ID, Name: "IT-Test Renamed", Description: "e"})
	if err != nil || upd.Name != "IT-Test Renamed" || upd.Description != "e" {
		t.Errorf("Update = %+v, %v", upd, err)
	}
	if _, err := repo.Update(ctx, &models.List{ID: 0, Name: "IT-Test none"}); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("Update unknown err = %v", err)
	}

	if err := repo.RemoveItem(ctx, l.ID, 900300); err != nil {
		t.Errorf("RemoveItem = %v", err)
	}
	if err := repo.RemoveItem(ctx, l.ID, 900300); !errors.Is(err, models.ErrListItemNotFound) {
		t.Errorf("second RemoveItem err = %v", err)
	}

	if _, err := repo.AddItem(ctx, l.ID, 900301); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, l.ID); err != nil {
		t.Errorf("Delete = %v", err)
	}
	if err := repo.Delete(ctx, l.ID); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("second Delete err = %v", err)
	}
	if _, err := repo.Get(ctx, l.ID); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("Get after delete err = %v", err)
	}
	var n int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM list_item WHERE list_id = $1`, l.ID).Scan(&n); err != nil || n != 0 {
		t.Errorf("items after cascade = %d, %v", n, err)
	}
}

// TestWebhookIntakeClaimOrdersPerTicket checks the claim query never hands out a ticket's second
// webhook while its first is open, but does let other tickets through.
func TestWebhookIntakeClaimOrdersPerTicket(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`) })
	repo := NewWebhookIntakeRepo(pool)

	first, err := repo.Insert(ctx, 900201, models.IntakeAdded, []byte(`{"a":1}`))
	if err != nil {
		t.Fatal(err)
	}
	second, _ := repo.Insert(ctx, 900201, models.IntakeUpdated, nil)
	other, _ := repo.Insert(ctx, 900202, models.IntakeUpdated, nil)

	c1, err := repo.Claim(ctx)
	if err != nil || c1.ID != first.ID {
		t.Fatalf("first claim = %+v, %v; want row %d", c1, err, first.ID)
	}
	if c1.Status != models.IntakeProcessing || c1.Attempts != 1 {
		t.Fatalf("claimed row status=%s attempts=%d", c1.Status, c1.Attempts)
	}

	// The same ticket's next row is blocked; the other ticket is not.
	c2, err := repo.Claim(ctx)
	if err != nil || c2.ID != other.ID {
		t.Fatalf("second claim = %+v, %v; want row %d", c2, err, other.ID)
	}
	if _, err := repo.Claim(ctx); !errors.Is(err, models.ErrIntakeEmpty) {
		t.Fatalf("third claim err = %v, want ErrIntakeEmpty", err)
	}

	// A rescheduled first row still blocks the second even though the second is ready now.
	if err := repo.Reschedule(ctx, first.ID, time.Now().Add(time.Hour), "boom"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Claim(ctx); !errors.Is(err, models.ErrIntakeEmpty) {
		t.Fatalf("claim behind a delayed sibling err = %v, want ErrIntakeEmpty", err)
	}

	// Once the first is done, the second is claimable.
	if err := repo.Finish(ctx, first.ID); err != nil {
		t.Fatal(err)
	}
	c3, err := repo.Claim(ctx)
	if err != nil || c3.ID != second.ID {
		t.Fatalf("claim after finish = %+v, %v; want row %d", c3, err, second.ID)
	}

	// Fail, retry and discard round-trip; retry and discard refuse non-failed rows.
	if err := repo.Fail(ctx, c3.ID, "gave up"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Discard(ctx, other.ID); !errors.Is(err, models.ErrIntakeNotFound) {
		t.Fatalf("discard of a processing row err = %v, want ErrIntakeNotFound", err)
	}
	retried, err := repo.Retry(ctx, c3.ID)
	if err != nil || retried.Status != models.IntakePending || retried.Attempts != 0 {
		t.Fatalf("retry = %+v, %v", retried, err)
	}

	// Crash recovery returns processing rows to pending.
	n, err := repo.ResetProcessing(ctx)
	if err != nil || n != 1 {
		t.Fatalf("reset = %d, %v; want 1", n, err)
	}

	st, err := repo.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.Counts[models.IntakePending] != 2 || st.Counts[models.IntakeDone] != 1 || st.LastReceivedAt == nil {
		t.Fatalf("stats = %+v", st)
	}
}

func TestWebhookIntakeCountsByHour(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`) })
	repo := NewWebhookIntakeRepo(pool)

	base := time.Now().Truncate(time.Hour).Add(-3 * time.Hour)
	for _, at := range []time.Time{base, base.Add(5 * time.Minute), base.Add(2 * time.Hour)} {
		if _, err := pool.Exec(ctx, `INSERT INTO webhook_intake (ticket_id, action, received_at) VALUES (1, 'updated', $1)`, at); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := repo.CountsByHour(ctx, base.Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Count != 2 || rows[1].Count != 1 || !rows[0].Hour.Equal(base) {
		t.Fatalf("rows = %+v", rows)
	}
}

func TestWorkflowRunRepoListFilters(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	seedTicket(t, pool, 900301, "Run summary ticket")
	seedTicket(t, pool, 900302, "Run summary ticket two")
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM workflow_run WHERE run_id LIKE 'test-run-%'`) })
	repo := NewWorkflowRunRepo(pool)

	// far in the past so DeleteBefore at the end cannot touch real rows in a shared test database
	base := time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)
	mk := func(id string, ticket, board int, at time.Time, outcome models.RunOutcome) {
		wfID := 7
		if err := repo.Insert(ctx, &models.WorkflowRun{RunID: id, TicketID: ticket, BoardID: board, WorkflowID: &wfID, WorkflowName: "wf",
			Event: models.TriggerUpdated, Source: models.SourceWebhook, StartedAt: at, Steps: 2, Outcome: outcome}); err != nil {
			t.Fatal(err)
		}
	}
	mk("test-run-1", 900301, 900001, base, models.OutcomeClean)
	mk("test-run-2", 900301, 900001, base.Add(time.Hour), models.OutcomeErrors)
	mk("test-run-3", 900302, 900009, base.Add(2*time.Hour), models.OutcomeClean)

	// the shared database holds real runs too, so order is checked inside the test rows' window
	lo, hi := base.Add(-time.Hour), base.Add(3*time.Hour)
	all, err := repo.List(ctx, models.RunFilter{From: &lo, To: &hi, Limit: 10})
	if err != nil || len(all) != 3 || all[0].RunID != "test-run-3" || all[2].RunID != "test-run-1" {
		t.Fatalf("list = %d rows, err %v, want test-run-3 first", len(all), err)
	}
	board := 900001
	byBoard, _ := repo.List(ctx, models.RunFilter{BoardID: &board, Limit: 10})
	if len(byBoard) != 2 {
		t.Errorf("board filter = %d, want 2", len(byBoard))
	}
	bad, _ := repo.List(ctx, models.RunFilter{Outcome: models.OutcomeErrors, Limit: 10})
	if len(bad) != 1 || bad[0].RunID != "test-run-2" {
		t.Errorf("outcome filter = %v", bad)
	}
	ticket := 900302
	byTicket, _ := repo.List(ctx, models.RunFilter{TicketID: &ticket, Limit: 10})
	if len(byTicket) != 1 {
		t.Errorf("ticket filter = %d, want 1", len(byTicket))
	}
	from, to := base.Add(30*time.Minute), base.Add(90*time.Minute)
	window, _ := repo.List(ctx, models.RunFilter{From: &from, To: &to, Limit: 10})
	if len(window) != 1 || window[0].RunID != "test-run-2" {
		t.Errorf("window filter = %v", window)
	}
	got, err := repo.Get(ctx, "test-run-2")
	if err != nil || got.Outcome != models.OutcomeErrors || got.WorkflowID == nil || *got.WorkflowID != 7 {
		t.Errorf("get = %+v, %v", got, err)
	}
	if _, err := repo.Get(ctx, "test-run-nope"); !errors.Is(err, models.ErrRunNotFound) {
		t.Errorf("missing run err = %v", err)
	}
	// insert is idempotent on run id (the backfill relies on it too)
	mk("test-run-1", 900301, 900001, base, models.OutcomeErrors)
	if again, _ := repo.Get(ctx, "test-run-1"); again.Outcome != models.OutcomeClean {
		t.Error("re-insert must not overwrite")
	}
	n, err := repo.DeleteBefore(ctx, base.Add(90*time.Minute))
	if err != nil || n < 2 {
		t.Errorf("delete before = %d, %v", n, err)
	}
}
