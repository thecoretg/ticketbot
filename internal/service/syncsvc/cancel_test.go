package syncsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

func TestStopNeedsAnUnfinishedRun(t *testing.T) {
	tr := &tracker{}
	if err := tr.stop("a@x", time.Now()); !errors.Is(err, ErrNoSyncRunning) {
		t.Fatalf("stop before any run = %v, want ErrNoSyncRunning", err)
	}

	_ = tr.start(&models.SyncPayload{CWBoards: true}, "", time.Now(), nil)
	tr.finish(time.Now())
	if err := tr.stop("a@x", time.Now()); !errors.Is(err, ErrNoSyncRunning) {
		t.Fatalf("stop after finish = %v, want ErrNoSyncRunning", err)
	}
}

func TestStopCancelsOnceAndKeepsTheFirstCanceller(t *testing.T) {
	tr := &tracker{}
	ctx, cancel := context.WithCancel(context.Background())
	_ = tr.start(&models.SyncPayload{CWTickets: true}, "starter@x", time.Now(), cancel)

	first := time.Now()
	if err := tr.stop("a@x", first); err != nil {
		t.Fatalf("stop: %v", err)
	}
	select {
	case <-ctx.Done():
	default:
		t.Fatal("stop did not cancel the run's context")
	}
	if err := tr.stop("b@x", first.Add(time.Second)); err != nil {
		t.Fatalf("second stop = %v, want nil", err)
	}

	run := tr.snapshot()
	if run.CancelledBy != "a@x" || run.CancelledAt == nil || !run.CancelledAt.Equal(first) {
		t.Errorf("cancelled = %q at %v, want the first stop's", run.CancelledBy, run.CancelledAt)
	}
	if !tr.running() {
		t.Error("running() = false before the phases wound down")
	}

	b, _ := json.Marshal(run)
	if !strings.Contains(string(b), `"cancelled_by":"a@x"`) {
		t.Errorf("run encodes %s, want cancelled_by", b)
	}

	tr.finish(time.Now())
	if tr.running() {
		t.Error("running() = true after finish")
	}
}

func TestStepIgnoresTheCancel(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWTickets: true}, "", time.Now(), nil)
	ph := tr.phase(models.SyncPhaseTickets)
	ph.counting("Processing tickets", 3)

	ph.step(nil)
	ph.step(fmt.Errorf("syncing ticket 2: %w", context.Canceled))

	got := tr.snapshot().Phases[0]
	if got.Done != 1 || got.ErrorCount != 0 || len(got.Errors) != 0 {
		t.Errorf("phase = %d done, %d errors %v; want the cancelled item neither counted nor listed", got.Done, got.ErrorCount, got.Errors)
	}
}

func TestEndPhase(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("boom")

	cases := []struct {
		name       string
		ctx        context.Context
		err        error
		rolledBack bool
		done       int
		total      int
		wantState  models.SyncPhaseState
		wantLabel  string
		wantErr    bool
	}{
		{"finished", context.Background(), nil, true, 5, 5, models.SyncPhaseDone, "Done", false},
		{"finished before the cancel reached it", cancelled, nil, true, 5, 5, models.SyncPhaseDone, "Done", false},
		{"failed", context.Background(), boom, true, 2, 5, models.SyncPhaseFailed, "Rolled back", true},
		{"cancelled in a transaction", cancelled, context.Canceled, true, 2, 5, models.SyncPhaseCancelled, "Cancelled, rolled back", false},
		{"cancelled mid tickets", cancelled, context.Canceled, false, 3, 10, models.SyncPhaseCancelled, "Cancelled after 3 of 10", false},
		{"cancelled while fetching", cancelled, context.Canceled, false, 0, 0, models.SyncPhaseCancelled, "Cancelled", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tr := &tracker{}
			_ = tr.start(&models.SyncPayload{CWTickets: true}, "", time.Now(), nil)
			ph := tr.phase(models.SyncPhaseTickets)
			if c.total > 0 {
				ph.counting("Processing tickets", c.total)
				for range c.done {
					ph.step(nil)
				}
			}

			err := endPhase(c.ctx, ph, c.err, "Rolled back", c.rolledBack)
			if (err != nil) != c.wantErr {
				t.Errorf("endPhase error = %v, want error %v", err, c.wantErr)
			}
			got := tr.snapshot().Phases[0]
			if got.State != c.wantState || got.Label != c.wantLabel {
				t.Errorf("phase = %q %q, want %q %q", got.State, got.Label, c.wantState, c.wantLabel)
			}
			if c.wantState == models.SyncPhaseCancelled && got.ErrorCount != 0 {
				t.Errorf("a cancelled phase lists %d errors, want none", got.ErrorCount)
			}
		})
	}
}

func TestForEachRunsEveryItemWithinTheLimit(t *testing.T) {
	for _, limit := range []int{0, 1, 3} {
		var ran, now, peak atomic.Int32
		items := make([]int, 20)
		err := forEach(context.Background(), items, limit, func(int) {
			n := now.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(time.Millisecond)
			now.Add(-1)
			ran.Add(1)
		})
		if err != nil {
			t.Errorf("limit %d: forEach = %v, want nil", limit, err)
		}
		if ran.Load() != 20 {
			t.Errorf("limit %d: ran %d of 20", limit, ran.Load())
		}
		if want := int32(max(limit, 1)); peak.Load() > want {
			t.Errorf("limit %d: %d ran at once", limit, peak.Load())
		}
	}
}

func TestForEachStopsStartingItemsOnCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var started, finished atomic.Int32
	items := make([]int, 100)
	for i := range items {
		items[i] = i
	}
	err := forEach(ctx, items, 2, func(i int) {
		started.Add(1)
		if i == 3 {
			cancel()
		}
		time.Sleep(5 * time.Millisecond)
		finished.Add(1)
	})

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("forEach = %v, want context.Canceled", err)
	}
	if n := started.Load(); n > 6 {
		t.Errorf("%d items started, want the loop to stop right after the cancel", n)
	}
	if started.Load() != finished.Load() {
		t.Errorf("returned with %d of %d started items unfinished", started.Load()-finished.Load(), started.Load())
	}
}

func TestForEachIsNotStoppedByACancelAfterTheLastStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var mu sync.Mutex
	var seen []int
	err := forEach(ctx, []int{1, 2, 3}, 1, func(i int) {
		mu.Lock()
		seen = append(seen, i)
		mu.Unlock()
		if i == 3 {
			cancel()
		}
	})
	if err != nil {
		t.Errorf("forEach = %v, want nil: every item had started", err)
	}
	if len(seen) != 3 {
		t.Errorf("ran %v, want all three", seen)
	}
}
