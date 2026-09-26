package syncsvc

import (
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

var ErrSyncRunning = errors.New("a sync is already running")

// tracker holds the running sync, or the last one, for GET /sync/status. It is shared by
// the Service and its withTx copies, so every phase reports into the same run.
type tracker struct {
	mu  sync.Mutex
	run *models.SyncRun
}

// start begins a run with one phase per selected sync, in page order. It fails while
// another run is unfinished, which is also what keeps two syncs from overlapping.
func (t *tracker) start(p *models.SyncPayload, startedBy string, now time.Time) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.run != nil && t.run.FinishedAt == nil {
		return ErrSyncRunning
	}

	run := &models.SyncRun{
		StartedAt: now,
		StartedBy: startedBy,
		Options: models.SyncOptions{
			CWBoards:        p.CWBoards,
			WebexRecipients: p.WebexRecipients,
			CWTickets:       p.CWTickets,
			BoardIDs:        slices.Clone(p.BoardIDs),
		},
	}
	add := func(on bool, name, label string) {
		if on {
			run.Phases = append(run.Phases, models.SyncPhase{Name: name, State: models.SyncPhaseFetching, Label: label})
		}
	}
	add(p.CWBoards, models.SyncPhaseBoards, "Fetching boards from ConnectWise")
	add(p.WebexRecipients, models.SyncPhaseWebexRecipients, "Fetching rooms from Webex")
	add(p.CWTickets, models.SyncPhaseTickets, "Fetching open tickets from ConnectWise")
	t.run = run
	return nil
}

func (t *tracker) finish(now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.run != nil {
		t.run.FinishedAt = &now
	}
}

func (t *tracker) running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.run != nil && t.run.FinishedAt == nil
}

// snapshot returns a copy of the run that the caller can encode without holding the lock.
func (t *tracker) snapshot() *models.SyncRun {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.run == nil {
		return nil
	}
	r := *t.run
	r.Options.BoardIDs = slices.Clone(r.Options.BoardIDs)
	if r.FinishedAt != nil {
		f := *r.FinishedAt
		r.FinishedAt = &f
	}
	r.Phases = slices.Clone(r.Phases)
	for i := range r.Phases {
		r.Phases[i].Errors = slices.Clone(r.Phases[i].Errors)
	}
	return &r
}

// phase returns the handle a sync reports through, or nil when the run has no such phase.
func (t *tracker) phase(name string) *phase {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.run == nil {
		return nil
	}
	for i := range t.run.Phases {
		if t.run.Phases[i].Name == name {
			return &phase{t: t, i: i}
		}
	}
	return nil
}

// phase is one phase of the current run. Every method is a no-op on a nil phase, so the
// sync functions can be called without a run.
type phase struct {
	t *tracker
	i int
}

func (p *phase) update(fn func(*models.SyncPhase)) {
	if p == nil {
		return
	}
	p.t.mu.Lock()
	defer p.t.mu.Unlock()
	fn(&p.t.run.Phases[p.i])
}

// fetching shows an indeterminate bar with label, for work whose size is not known yet.
func (p *phase) fetching(label string) {
	p.update(func(ph *models.SyncPhase) {
		ph.State, ph.Label, ph.Done, ph.Total = models.SyncPhaseFetching, label, 0, 0
	})
}

// counting starts the bar at 0 of total.
func (p *phase) counting(label string, total int) {
	p.update(func(ph *models.SyncPhase) {
		ph.State, ph.Label, ph.Done, ph.Total = models.SyncPhaseRunning, label, 0, total
	})
}

// label changes the text under the bar and leaves the count alone.
func (p *phase) label(label string) {
	p.update(func(ph *models.SyncPhase) { ph.Label = label })
}

// step counts one item as processed; a non-nil err is recorded against it.
func (p *phase) step(err error) {
	p.update(func(ph *models.SyncPhase) {
		ph.Done++
		if err != nil {
			addError(ph, err)
		}
	})
}

func (p *phase) done() {
	p.update(func(ph *models.SyncPhase) {
		ph.State, ph.Label = models.SyncPhaseDone, "Done"
	})
}

// fail ends the phase with label and records err, the reason it stopped.
func (p *phase) fail(label string, err error) {
	p.update(func(ph *models.SyncPhase) {
		ph.State, ph.Label = models.SyncPhaseFailed, label
		addError(ph, err)
	})
}

func addError(ph *models.SyncPhase, err error) {
	ph.ErrorCount++
	if len(ph.Errors) < models.MaxSyncPhaseErrors {
		ph.Errors = append(ph.Errors, err.Error())
	}
}
