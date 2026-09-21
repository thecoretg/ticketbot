package workflow

import (
	"testing"
	"time"
)

func TestRateLimiterBlocksAtTheCapAndLifts(t *testing.T) {
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	now := base
	l := NewRateLimiter(func() int { return 3 })
	l.now = func() time.Time { return now }
	var blocked []int
	l.OnBlock = func(id int, _ time.Time) { blocked = append(blocked, id) }

	for i := 0; i < 3; i++ {
		if err := l.Reserve(1, 1); err != nil {
			t.Fatalf("write %d refused: %v", i+1, err)
		}
	}
	if err := l.Reserve(1, 1); err == nil {
		t.Fatal("fourth write within the window should be refused")
	}
	if len(blocked) != 1 || blocked[0] != 1 {
		t.Fatalf("OnBlock calls = %v, want [1]", blocked)
	}
	// Other tickets are unaffected.
	if err := l.Reserve(2, 1); err != nil {
		t.Fatalf("other ticket refused: %v", err)
	}
	// Still blocked inside the block period even though the window has rolled.
	now = base.Add(30 * time.Minute)
	if err := l.Reserve(1, 1); err == nil {
		t.Fatal("blocked ticket should stay refused inside the block")
	}
	if len(blocked) != 1 {
		t.Fatal("a refused write while blocked must not re-alert")
	}
	now = base.Add(61 * time.Minute)
	if err := l.Reserve(1, 1); err != nil {
		t.Fatalf("after the block lifts: %v", err)
	}
}

func TestRateLimiterWindowSlides(t *testing.T) {
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	now := base
	l := NewRateLimiter(func() int { return 2 })
	l.now = func() time.Time { return now }

	_ = l.Reserve(1, 2)
	now = base.Add(16 * time.Minute)
	if err := l.Reserve(1, 1); err != nil {
		t.Fatalf("writes older than the window should not count: %v", err)
	}
}

func TestRateLimiterDisabledAtZero(t *testing.T) {
	l := NewRateLimiter(func() int { return 0 })
	for i := 0; i < 100; i++ {
		if err := l.Reserve(1, 1); err != nil {
			t.Fatal(err)
		}
	}
}
