package intake

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/models"
)

// fakeRepo is an in-memory queue. Claim ignores per-ticket ordering, which the Postgres query owns
// and the integration test covers; these tests are about what the worker does with a row.
type fakeRepo struct {
	repos.WebhookIntakeRepository
	mu   sync.Mutex
	rows []*models.WebhookIntake
	seq  int64
}

func (f *fakeRepo) Insert(_ context.Context, ticketID int, action models.IntakeAction, payload []byte) (*models.WebhookIntake, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	r := &models.WebhookIntake{ID: f.seq, TicketID: ticketID, Action: action, Payload: payload, Status: models.IntakePending, NextAttemptAt: time.Now()}
	f.rows = append(f.rows, r)
	return r, nil
}

func (f *fakeRepo) Claim(context.Context) (*models.WebhookIntake, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.Status == models.IntakePending && !r.NextAttemptAt.After(time.Now()) {
			r.Status = models.IntakeProcessing
			r.Attempts++
			c := *r
			return &c, nil
		}
	}
	return nil, models.ErrIntakeEmpty
}

func (f *fakeRepo) set(id int64, fn func(*models.WebhookIntake)) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ID == id {
			fn(r)
			return nil
		}
	}
	return errors.New("no row")
}

func (f *fakeRepo) Finish(_ context.Context, id int64) error {
	return f.set(id, func(r *models.WebhookIntake) { r.Status = models.IntakeDone })
}

func (f *fakeRepo) Reschedule(_ context.Context, id int64, at time.Time, lastError string) error {
	return f.set(id, func(r *models.WebhookIntake) {
		r.Status = models.IntakePending
		r.NextAttemptAt = at
		r.LastError = &lastError
	})
}

func (f *fakeRepo) Fail(_ context.Context, id int64, lastError string) error {
	return f.set(id, func(r *models.WebhookIntake) { r.Status = models.IntakeFailed; r.LastError = &lastError })
}

func (f *fakeRepo) ResetProcessing(context.Context) (int64, error) { return 0, nil }

func (f *fakeRepo) DeleteFinishedBefore(context.Context, time.Time) (int64, error) { return 0, nil }

func (f *fakeRepo) get(id int64) models.WebhookIntake {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ID == id {
			return *r
		}
	}
	return models.WebhookIntake{}
}

type fakeProcessor struct {
	mu      sync.Mutex
	calls   []string
	fail    int // fail this many calls before succeeding
	deletes int
}

func (p *fakeProcessor) ProcessTicket(_ context.Context, id int, _ ticketbot.ProcessOpts) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.calls = append(p.calls, "process")
	if p.fail > 0 {
		p.fail--
		return errors.New("connectwise unavailable")
	}
	return nil
}

func (p *fakeProcessor) SoftDeleteTicket(context.Context, int, models.EventSource) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.deletes++
	return nil
}

func (p *fakeProcessor) count() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.calls)
}

type fakeAlerter struct {
	mu       sync.Mutex
	subjects []string
}

func (a *fakeAlerter) Alert(_ context.Context, subject, _ string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.subjects = append(a.subjects, subject)
}

func newTestService(repo *fakeRepo, proc *fakeProcessor, alerter *fakeAlerter, backoff []time.Duration) *Service {
	s := New(Params{Repo: repo, Processor: proc, Alerter: alerter})
	s.Backoff = backoff
	return s
}

func TestProcessSuccessMarksDone(t *testing.T) {
	repo, proc := &fakeRepo{}, &fakeProcessor{}
	s := newTestService(repo, proc, &fakeAlerter{}, DefaultBackoff)
	row, _ := repo.Insert(context.Background(), 42, models.IntakeUpdated, nil)
	claimed, _ := repo.Claim(context.Background())

	s.process(context.Background(), claimed)

	if got := repo.get(row.ID); got.Status != models.IntakeDone {
		t.Fatalf("status = %s, want done", got.Status)
	}
	if proc.count() != 1 {
		t.Fatalf("processor called %d times, want 1", proc.count())
	}
}

func TestProcessFailureReschedulesWithBackoff(t *testing.T) {
	repo, proc := &fakeRepo{}, &fakeProcessor{fail: 10}
	backoff := []time.Duration{time.Minute, time.Hour}
	s := newTestService(repo, proc, &fakeAlerter{}, backoff)
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return base }
	row, _ := repo.Insert(context.Background(), 42, models.IntakeUpdated, nil)

	claimed, _ := repo.Claim(context.Background())
	s.process(context.Background(), claimed)
	got := repo.get(row.ID)
	if got.Status != models.IntakePending || !got.NextAttemptAt.Equal(base.Add(time.Minute)) {
		t.Fatalf("after attempt 1: status=%s next=%v, want pending at +1m", got.Status, got.NextAttemptAt)
	}
	if got.LastError == nil || *got.LastError == "" {
		t.Fatal("last_error not recorded")
	}

	// Second attempt uses the second delay.
	_ = repo.set(row.ID, func(r *models.WebhookIntake) { r.NextAttemptAt = base })
	claimed, _ = repo.Claim(context.Background())
	s.process(context.Background(), claimed)
	got = repo.get(row.ID)
	if got.Status != models.IntakePending || !got.NextAttemptAt.Equal(base.Add(time.Hour)) {
		t.Fatalf("after attempt 2: status=%s next=%v, want pending at +1h", got.Status, got.NextAttemptAt)
	}
}

func TestProcessExhaustedFailsAndAlerts(t *testing.T) {
	repo, proc, alerter := &fakeRepo{}, &fakeProcessor{fail: 10}, &fakeAlerter{}
	s := newTestService(repo, proc, alerter, []time.Duration{time.Millisecond})
	row, _ := repo.Insert(context.Background(), 42, models.IntakeUpdated, nil)

	for i := 0; i < 2; i++ {
		_ = repo.set(row.ID, func(r *models.WebhookIntake) { r.NextAttemptAt = time.Now().Add(-time.Second) })
		claimed, err := repo.Claim(context.Background())
		if err != nil {
			t.Fatalf("claim %d: %v", i, err)
		}
		s.process(context.Background(), claimed)
	}

	got := repo.get(row.ID)
	if got.Status != models.IntakeFailed {
		t.Fatalf("status = %s, want failed", got.Status)
	}
	if got.Attempts != 2 {
		t.Fatalf("attempts = %d, want 2", got.Attempts)
	}
	if len(alerter.subjects) != 1 {
		t.Fatalf("alerts = %d, want 1", len(alerter.subjects))
	}
}

func TestDeletedActionSoftDeletes(t *testing.T) {
	repo, proc := &fakeRepo{}, &fakeProcessor{}
	s := newTestService(repo, proc, &fakeAlerter{}, DefaultBackoff)
	_, _ = repo.Insert(context.Background(), 7, models.IntakeDeleted, nil)
	claimed, _ := repo.Claim(context.Background())

	s.process(context.Background(), claimed)

	if proc.deletes != 1 || proc.count() != 0 {
		t.Fatalf("deletes=%d processes=%d, want 1/0", proc.deletes, proc.count())
	}
}

func TestWorkerDrainsEnqueuedRows(t *testing.T) {
	repo, proc := &fakeRepo{}, &fakeProcessor{}
	s := newTestService(repo, proc, &fakeAlerter{}, DefaultBackoff)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.Start(ctx)

	for i := 1; i <= 3; i++ {
		if _, err := s.Enqueue(ctx, i, models.IntakeAdded, []byte(`{}`)); err != nil {
			t.Fatal(err)
		}
	}

	deadline := time.Now().Add(2 * time.Second)
	for proc.count() < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if proc.count() != 3 {
		t.Fatalf("processed %d rows, want 3", proc.count())
	}
	for i := int64(1); i <= 3; i++ {
		if got := repo.get(i); got.Status != models.IntakeDone {
			t.Errorf("row %d status = %s, want done", i, got.Status)
		}
	}

	cancel()
	waitCtx, waitCancel := context.WithTimeout(context.Background(), time.Second)
	defer waitCancel()
	s.Wait(waitCtx)
}
