package workflow

import (
	"fmt"
	"sync"
	"time"
)

// RateLimiter caps ConnectWise writes per ticket over a sliding window. A ticket that reaches the
// cap is blocked for Block; writes attempted while blocked are refused with an error naming when
// the block lifts. It is in-memory: a restart clears it, which is acceptable for a guard whose
// job is to stop a runaway loop, not to account precisely.
type RateLimiter struct {
	// Limit returns the current cap; it is read on every reservation so config changes apply
	// without a restart. A limit of 0 or less disables the cap.
	Limit  func() int
	Window time.Duration
	Block  time.Duration
	// OnBlock is called once when a ticket is blocked, with the ticket and when the block lifts.
	OnBlock func(ticketID int, until time.Time)

	now func() time.Time
	mu  sync.Mutex
	per map[int]*ticketWrites
}

type ticketWrites struct {
	times        []time.Time
	blockedUntil time.Time
}

// Defaults for the cap: 20 writes per ticket per rolling 15 minutes, then an hour's block. A
// legitimate run makes one PATCH plus one call per note, so 20 is far above normal traffic.
const (
	DefaultWriteWindow = 15 * time.Minute
	DefaultWriteBlock  = time.Hour
	DefaultWriteLimit  = 20
)

func NewRateLimiter(limit func() int) *RateLimiter {
	return &RateLimiter{Limit: limit, Window: DefaultWriteWindow, Block: DefaultWriteBlock, now: time.Now, per: map[int]*ticketWrites{}}
}

// Reserve records n writes for the ticket, or refuses them.
func (l *RateLimiter) Reserve(ticketID int, n int) error {
	limit := DefaultWriteLimit
	if l.Limit != nil {
		limit = l.Limit()
	}
	if limit <= 0 {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()

	tw := l.per[ticketID]
	if tw == nil {
		tw = &ticketWrites{}
		l.per[ticketID] = tw
	}
	if now.Before(tw.blockedUntil) {
		return l.blockedErr(tw.blockedUntil)
	}

	cutoff := now.Add(-l.Window)
	kept := tw.times[:0]
	for _, t := range tw.times {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	tw.times = kept

	if len(tw.times)+n > limit {
		tw.blockedUntil = now.Add(l.Block)
		tw.times = nil
		if l.OnBlock != nil {
			l.OnBlock(ticketID, tw.blockedUntil)
		}
		return l.blockedErr(tw.blockedUntil)
	}
	for i := 0; i < n; i++ {
		tw.times = append(tw.times, now)
	}

	// Forget tickets that have gone quiet so the map does not grow with every ticket ever seen.
	for id, other := range l.per {
		if id != ticketID && len(other.times) == 0 && !now.Before(other.blockedUntil) {
			delete(l.per, id)
		}
	}
	return nil
}

func (l *RateLimiter) blockedErr(until time.Time) error {
	return fmt.Errorf("write cap reached for this ticket; writes blocked until %s", until.Format(time.RFC3339))
}
