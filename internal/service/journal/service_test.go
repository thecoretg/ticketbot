package journal

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type fakeRepo struct {
	store map[int]*models.TicketJournal
}

func newFakeRepo() *fakeRepo { return &fakeRepo{store: map[int]*models.TicketJournal{}} }

func (f *fakeRepo) WithTx(tx pgx.Tx) repos.TicketJournalRepository { return f }

func (f *fakeRepo) Get(ctx context.Context, id int) (*models.TicketJournal, error) {
	j, ok := f.store[id]
	if !ok {
		return nil, models.ErrTicketJournalNotFound
	}
	cp := *j
	cp.Runs = append([]models.TicketRun(nil), j.Runs...)
	return &cp, nil
}

func (f *fakeRepo) ListSummaries(ctx context.Context, limit int) ([]*models.TicketJournal, error) {
	return nil, nil
}

func (f *fakeRepo) Upsert(ctx context.Context, j *models.TicketJournal) (*models.TicketJournal, error) {
	cp := *j
	f.store[j.TicketID] = &cp
	return j, nil
}

func (f *fakeRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}

func sampleFull(summary string) *models.FullTicket {
	ft := &models.FullTicket{}
	ft.Ticket.ID = 42
	ft.Ticket.Summary = summary
	ft.Board.Name = "Service Board"
	ft.Company.Name = "Acme Corp"
	ft.Status.Name = "New"
	return ft
}

func run(t time.Time, outcome string) models.TicketRun {
	return models.TicketRun{Time: t, Trigger: models.TriggerUpdated, Outcome: outcome}
}

func TestRecordAppendsAndSnapshots(t *testing.T) {
	s := New(newFakeRepo(), &models.Config{})
	ctx := t.Context()

	if err := s.Record(ctx, 42, sampleFull("Printer broken"), run(time.Now(), models.OutcomeCompleted)); err != nil {
		t.Fatal(err)
	}
	j, err := s.Get(ctx, 42)
	if err != nil {
		t.Fatal(err)
	}
	if j.Summary != "Printer broken" || j.CompanyName != "Acme Corp" || j.StatusName != "New" {
		t.Fatalf("snapshot not set: %+v", j)
	}
	if len(j.Runs) != 1 || j.LastOutcome != models.OutcomeCompleted {
		t.Fatalf("expected 1 run with outcome, got %d runs outcome=%q", len(j.Runs), j.LastOutcome)
	}
}

func TestRecordNilFullKeepsSnapshot(t *testing.T) {
	s := New(newFakeRepo(), &models.Config{})
	ctx := t.Context()

	_ = s.Record(ctx, 42, sampleFull("Original summary"), run(time.Now(), models.OutcomeCompleted))
	// A later run with no synced ticket (e.g. sync failed) must not wipe the snapshot.
	_ = s.Record(ctx, 42, nil, run(time.Now(), models.OutcomeWithErrors))

	j, _ := s.Get(ctx, 42)
	if j.Summary != "Original summary" {
		t.Fatalf("nil full should keep prior summary, got %q", j.Summary)
	}
	if len(j.Runs) != 2 || j.LastOutcome != models.OutcomeWithErrors {
		t.Fatalf("expected 2 runs, last outcome with-errors; got %d / %q", len(j.Runs), j.LastOutcome)
	}
}

func TestRecordCapsRuns(t *testing.T) {
	s := New(newFakeRepo(), &models.Config{})
	ctx := t.Context()

	for range maxRunsPerTicket + 5 {
		if err := s.Record(ctx, 42, sampleFull("x"), run(time.Now(), models.OutcomeCompleted)); err != nil {
			t.Fatal(err)
		}
	}
	j, _ := s.Get(ctx, 42)
	if len(j.Runs) != maxRunsPerTicket {
		t.Fatalf("expected runs capped at %d, got %d", maxRunsPerTicket, len(j.Runs))
	}
}
