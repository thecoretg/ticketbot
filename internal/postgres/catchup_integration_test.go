package postgres

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// TestCatchupQueries covers what the missed-webhook check reads: ConnectWise's lastUpdated off
// the raw copy, which tickets have an open intake row, and that catch-up rows never count as a
// received webhook.
func TestCatchupQueries(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	_, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM webhook_intake`) })

	seedTicket(t, pool, 900301, "has lastUpdated")
	seedTicket(t, pool, 900302, "no _info in raw")
	raw, _ := json.Marshal(map[string]any{"id": 900301, "_info": map[string]any{"lastUpdated": "2026-09-24T14:24:58.253Z"}})
	if _, err := pool.Exec(ctx, `UPDATE cw_ticket SET raw = $2 WHERE id = $1`, 900301, raw); err != nil {
		t.Fatal(err)
	}

	got, err := NewTicketRepo(pool).CWLastUpdated(ctx, []int{900301, 900302, 900399})
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, 9, 24, 14, 24, 58, 253e6, time.UTC)
	if len(got) != 2 || !got[900301].Equal(want) || !got[900302].Equal(time.Unix(0, 0)) {
		t.Fatalf("CWLastUpdated = %v, want 900301 at %v and 900302 at the epoch", got, want)
	}

	repo := NewWebhookIntakeRepo(pool)
	webhook, _ := repo.Insert(ctx, 900301, models.IntakeUpdated, nil)
	if _, err := repo.Insert(ctx, 900302, models.IntakeCatchup, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, webhook.ID); err != nil {
		t.Fatal(err)
	}

	open, err := repo.OpenTickets(ctx, []int{900301, 900302})
	if err != nil {
		t.Fatal(err)
	}
	if len(open) != 1 || open[0] != 900302 {
		t.Errorf("OpenTickets = %v, want [900302]", open)
	}

	n, err := repo.CountActionSince(ctx, models.IntakeCatchup, time.Now().Add(-time.Hour))
	if err != nil || n != 1 {
		t.Errorf("CountActionSince = %d, %v; want 1", n, err)
	}

	st, err := repo.Stats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if st.LastReceivedAt == nil || !st.LastReceivedAt.Equal(webhook.ReceivedAt) {
		t.Errorf("LastReceivedAt = %v, want the webhook's %v, not the later catch-up row", st.LastReceivedAt, webhook.ReceivedAt)
	}
}

func TestCatchupStateRepo(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewCatchupStateRepo(pool)

	// the table is a singleton the running app also uses; put back whatever was there
	before, err := repo.Get(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if before == nil {
			_, _ = pool.Exec(ctx, `DELETE FROM webhook_catchup`)
			return
		}
		_ = repo.Save(ctx, before)
	})

	mark := time.Now().UTC().Truncate(time.Microsecond)
	msg := "connectwise down"
	for _, st := range []models.CatchupState{
		{CheckedThrough: &mark, LastRunAt: &mark, LastQueued: 3},
		{CheckedThrough: &mark, LastRunAt: &mark, LastError: &msg},
	} {
		if err := repo.Save(ctx, &st); err != nil {
			t.Fatal(err)
		}
		got, err := repo.Get(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if got == nil || !got.CheckedThrough.Equal(mark) || got.LastQueued != st.LastQueued ||
			(got.LastError == nil) != (st.LastError == nil) {
			t.Errorf("Get = %+v, want %+v", got, st)
		}
	}
}
