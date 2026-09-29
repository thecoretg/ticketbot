package syncsvc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/alerts"
	"github.com/thecoretg/ticketbot/models"
)

const (
	// NightlyJobName is the nightly sync's row in scheduled_job.
	NightlyJobName = "nightly_sync"
	// NightlyStartedBy is what the Sync page shows as who started it.
	NightlyStartedBy = "Nightly schedule"
	// nightlyWindow is how late a missed run may still start: an instance that was down at the
	// scheduled time runs when it comes back, unless that is well into the day, when a full
	// sync would compete with daytime webhooks. Then it waits for the next night.
	nightlyWindow = 3 * time.Hour
	nightlyCheck  = time.Minute
)

// Syncer is the part of Service the schedule drives.
type Syncer interface {
	IsSyncing() bool
	RunAndWait(ctx context.Context, payload *models.SyncPayload, startedBy string) error
}

// Callbacks re-registers the ConnectWise ticket callback. *webhooks.Service satisfies it.
type Callbacks interface {
	EnsureTicketCallback(ctx context.Context) (added bool, err error)
}

// Nightly runs the full reconcile once a day: every sync phase, open tickets on the boards that
// have an enabled workflow, and the ticket callback. Failures go to the ops room.
type Nightly struct {
	Sync Syncer
	// Callbacks is nil where webhooks are not registered (SKIP_HOOKS), and the check is skipped.
	Callbacks Callbacks
	Workflows interface {
		List(ctx context.Context) ([]*models.Workflow, error)
	}
	Jobs    repos.ScheduledJobRepository
	Cfg     *models.Config
	Alerter alerts.Alerter

	now func() time.Time
}

func NewNightly(n Nightly) *Nightly {
	n.now = time.Now
	if n.Alerter == nil {
		n.Alerter = alerts.Log{}
	}
	return &n
}

// Start launches the schedule loop; it stops when ctx is cancelled.
func (n *Nightly) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(nightlyCheck)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				n.tick(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (n *Nightly) tick(ctx context.Context) {
	job, err := n.Jobs.Get(ctx, NightlyJobName)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("nightly sync: reading its record", "error", err.Error())
		}
		return
	}
	if _, ok := n.due(n.now(), job); !ok {
		return
	}
	if n.Sync.IsSyncing() {
		slog.Info("nightly sync: a sync is already running; trying again in a minute")
		return
	}
	n.Run(ctx)
}

// due reports whether today's run should start now, and when it was scheduled.
func (n *Nightly) due(now time.Time, job *models.ScheduledJob) (time.Time, bool) {
	if !n.Cfg.NightlySyncEnabled {
		return time.Time{}, false
	}
	at, err := n.scheduledAt(now)
	if err != nil {
		return time.Time{}, false
	}
	if now.Before(at) || now.After(at.Add(nightlyWindow)) {
		return at, false
	}
	if job != nil && job.LastStartedAt != nil && !job.LastStartedAt.Before(at) {
		return at, false // today's run already started
	}
	return at, true
}

// scheduledAt is today's run time in the business time zone.
func (n *Nightly) scheduledAt(now time.Time) (time.Time, error) {
	hm, err := time.Parse("15:04", n.Cfg.NightlySyncTime)
	if err != nil {
		return time.Time{}, fmt.Errorf("nightly sync time %q: %w", n.Cfg.NightlySyncTime, err)
	}
	loc, err := time.LoadLocation(n.Cfg.BusinessZone)
	if err != nil {
		loc = time.UTC
	}
	local := now.In(loc)
	return time.Date(local.Year(), local.Month(), local.Day(), hm.Hour(), hm.Minute(), 0, 0, loc), nil
}

// Run does one nightly reconcile and records it. The callback goes first, so a lost callback is
// restored before the long ticket phase starts.
func (n *Nightly) Run(ctx context.Context) {
	start := n.now()
	job := &models.ScheduledJob{Name: NightlyJobName, LastStartedAt: &start}
	if err := n.Jobs.Save(ctx, job); err != nil {
		slog.Warn("nightly sync: recording its start", "error", err.Error())
	}
	slog.Info("nightly sync: starting")

	var errs []error
	if n.Callbacks != nil {
		added, err := n.Callbacks.EnsureTicketCallback(ctx)
		switch {
		case err != nil:
			errs = append(errs, fmt.Errorf("checking the connectwise ticket callback: %w", err))
		case added:
			n.Alerter.Alert(ctx, "ConnectWise ticket callback was missing",
				"The nightly check found no ticket callback for this instance in ConnectWise and registered it again. "+
					"Webhooks were not arriving before this; the missed-webhook check covers the last 24 hours.")
		}
	}

	boards, err := n.watchedBoards(ctx)
	if err != nil {
		errs = append(errs, err)
	}
	payload := &models.SyncPayload{
		CWBoards:           true,
		WebexRecipients:    true,
		CWMembers:          true,
		CWCompanies:        true,
		CWContacts:         true,
		CWTickets:          len(boards) > 0,
		BoardIDs:           boards,
		RunWorkflows:       n.Cfg.NightlySyncRunWorkflows,
		MaxConcurrentSyncs: n.Cfg.MaxConcurrentSyncs,
	}
	if err := n.Sync.RunAndWait(ctx, payload, NightlyStartedBy); err != nil {
		errs = append(errs, err)
	}

	finished := n.now()
	job.LastFinishedAt = &finished
	if len(errs) > 0 {
		msg := errors.Join(errs...).Error()
		job.LastError = &msg
	}
	if err := n.Jobs.Save(context.WithoutCancel(ctx), job); err != nil {
		slog.Warn("nightly sync: recording its finish", "error", err.Error())
	}

	if len(errs) == 0 {
		slog.Info("nightly sync: finished", "took", finished.Sub(start).Round(time.Second).String())
		return
	}
	if ctx.Err() != nil {
		return // shutting down; the errors are the cancel, not failures
	}
	slog.Error("nightly sync: finished with errors", "error", *job.LastError)
	n.Alerter.Alert(ctx, "Nightly sync finished with errors", summarize(errs)+"\nThe Sync page lists each phase's errors.")
}

// watchedBoards is every board with an enabled workflow, the only boards whose open tickets the
// nightly run fetches.
func (n *Nightly) watchedBoards(ctx context.Context) ([]int, error) {
	wfs, err := n.Workflows.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing workflows: %w", err)
	}
	var boards []int
	for _, w := range wfs {
		if w.Enabled && !slices.Contains(boards, w.BoardID) {
			boards = append(boards, w.BoardID)
		}
	}
	slices.Sort(boards)
	return boards, nil
}

// summarize keeps an alert short: the first line of each error, at most five.
func summarize(errs []error) string {
	var lines []string
	for _, e := range errs {
		for l := range strings.SplitSeq(e.Error(), "\n") {
			if len(lines) == 5 {
				return strings.Join(lines, "\n") + "\n…"
			}
			lines = append(lines, l)
		}
	}
	return strings.Join(lines, "\n")
}
