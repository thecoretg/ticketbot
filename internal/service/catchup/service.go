// Package catchup finds tickets ConnectWise changed without sending a webhook and queues them on
// the intake queue as if the webhook had come, so their workflows run and their notifications go
// out, one interval late.
//
// Each run is one ConnectWise list request for tickets on boards with an enabled workflow whose
// lastUpdated is after the watermark. Only tickets whose lastUpdated is newer than the stored copy,
// and that have nothing waiting in intake, are queued; a ticket whose webhook did arrive matches
// and is skipped, so a run that finds nothing costs that one request.
package catchup

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

const (
	// checkEvery is how often the loop looks at whether a run is due. The interval itself comes
	// from config so a change applies without a restart.
	checkEvery = time.Minute
	// overlap reaches back past the watermark to cover clock skew between ConnectWise and this
	// server and any delay before an update shows in ConnectWise search. Tickets seen twice match
	// their stored copy the second time, so the overlap costs nothing.
	overlap = 5 * time.Minute
	// maxLookback caps the window after long downtime; the nightly reconcile covers anything older.
	maxLookback = 24 * time.Hour
)

// Lister is the ConnectWise call a run makes. *psa.Client satisfies it.
type Lister interface {
	ListTickets(ctx context.Context, params map[string]string, opts ...psa.ListOption) ([]psa.Ticket, error)
}

// Queue is where missed tickets go. *intake.Service satisfies it.
type Queue interface {
	Enqueue(ctx context.Context, ticketID int, action models.IntakeAction, payload []byte) (*models.WebhookIntake, error)
}

type Config interface {
	GetCatchupIntervalMinutes() int
}

// OpenRows reports which tickets already have a webhook waiting or running.
type OpenRows interface {
	OpenTickets(ctx context.Context, ticketIDs []int) ([]int, error)
}

// StoredTickets reads ConnectWise's lastUpdated off each stored ticket.
type StoredTickets interface {
	CWLastUpdated(ctx context.Context, ids []int) (map[int]time.Time, error)
}

type WorkflowLister interface {
	List(ctx context.Context) ([]*models.Workflow, error)
}

type Service struct {
	CW        Lister
	Queue     Queue
	Intake    OpenRows
	Tickets   StoredTickets
	Workflows WorkflowLister
	State     repos.CatchupStateRepository
	Cfg       Config

	now func() time.Time
}

type Params struct {
	CW        Lister
	Queue     Queue
	Intake    OpenRows
	Tickets   StoredTickets
	Workflows WorkflowLister
	State     repos.CatchupStateRepository
	Cfg       Config
}

func New(p Params) *Service {
	return &Service{CW: p.CW, Queue: p.Queue, Intake: p.Intake, Tickets: p.Tickets, Workflows: p.Workflows,
		State: p.State, Cfg: p.Cfg, now: time.Now}
}

// Start launches the loop. It stops when ctx is cancelled; a run cut short leaves the watermark
// where it was, so the next process repeats it.
func (s *Service) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(checkEvery)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s.tick(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// tick runs the check when the configured interval has passed since the last run, successful or
// not, so a ConnectWise outage is retried at the normal pace rather than every minute.
func (s *Service) tick(ctx context.Context) {
	minutes := s.Cfg.GetCatchupIntervalMinutes()
	if minutes <= 0 {
		return
	}
	st, err := s.State.Get(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("catchup: reading state", "error", err.Error())
		}
		return
	}
	interval := time.Duration(minutes) * time.Minute
	if st != nil && st.LastRunAt != nil && s.now().Sub(*st.LastRunAt) < interval {
		return
	}
	if _, err := s.Run(ctx, st, interval); err != nil && ctx.Err() == nil {
		slog.Warn("catchup: missed-webhook check failed; the next run retries the same window", "error", err.Error())
	}
}

// Run does one check and records it. It returns how many tickets it queued. st is the state
// before the run (nil before the first); interval sizes the first run's window.
func (s *Service) Run(ctx context.Context, st *models.CatchupState, interval time.Duration) (int, error) {
	start := s.now()
	queued, err := s.check(ctx, st, start, interval)

	next := models.CatchupState{LastRunAt: &start, LastQueued: queued}
	if st != nil {
		next.CheckedThrough = st.CheckedThrough
	}
	if err != nil {
		msg := err.Error()
		next.LastError = &msg
	} else {
		next.CheckedThrough = &start
	}
	if serr := s.State.Save(ctx, &next); serr != nil && err == nil {
		err = fmt.Errorf("saving catch-up state: %w", serr)
	}
	return queued, err
}

func (s *Service) check(ctx context.Context, st *models.CatchupState, start time.Time, interval time.Duration) (int, error) {
	since := start.Add(-interval)
	if st != nil && st.CheckedThrough != nil {
		since = *st.CheckedThrough
	}
	if floor := start.Add(-maxLookback); since.Before(floor) {
		slog.Warn("catchup: last successful check is older than the lookback cap; checking the last 24 hours only",
			"checked_through", since, "lookback", maxLookback.String())
		since = floor
	}
	from := since.Add(-overlap)

	boards, err := s.watchedBoards(ctx)
	if err != nil {
		return 0, err
	}
	if len(boards) == 0 {
		return 0, nil
	}

	tix, err := s.CW.ListTickets(ctx, map[string]string{
		"conditions": conditions(from, boards),
		"fields":     "id,_info/lastUpdated,_info/dateEntered",
		"pageSize":   "1000",
	})
	if err != nil {
		return 0, fmt.Errorf("listing changed tickets from connectwise: %w", err)
	}
	if len(tix) == 0 {
		return 0, nil
	}

	ids := make([]int, len(tix))
	for i, t := range tix {
		ids[i] = t.ID
	}
	stored, err := s.Tickets.CWLastUpdated(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("reading stored tickets: %w", err)
	}
	openIDs, err := s.Intake.OpenTickets(ctx, ids)
	if err != nil {
		return 0, fmt.Errorf("reading open intake rows: %w", err)
	}

	var missed []int
	for _, t := range tix {
		if slices.Contains(openIDs, t.ID) {
			continue // its webhook is queued or running; that run will see the change
		}
		have, ok := stored[t.ID]
		switch {
		case ok && !t.Info.LastUpdated.After(have):
			continue // processed already
		case !ok && t.Info.DateEntered.Before(from):
			// An older ticket ticketbot never stored, such as one on a board that only just got a
			// workflow. Queuing it would fire "created" triggers for a ticket that is not new.
			slog.Debug("catchup: skipping unstored ticket entered before the window", "ticket_id", t.ID)
			continue
		}
		payload, _ := json.Marshal(map[string]any{"cw_last_updated": t.Info.LastUpdated, "stored_last_updated": storedOrNil(have, ok)})
		if _, err := s.Queue.Enqueue(ctx, t.ID, models.IntakeCatchup, payload); err != nil {
			return len(missed), fmt.Errorf("queueing ticket %d: %w", t.ID, err)
		}
		missed = append(missed, t.ID)
	}

	if len(missed) > 0 {
		slog.Info("catchup: queued tickets ConnectWise changed without a webhook", "count", len(missed), "ticket_ids", missed)
	} else {
		slog.Debug("catchup: no missed webhooks", "checked", len(tix))
	}
	return len(missed), nil
}

// watchedBoards is every board with an enabled workflow: a missed webhook anywhere else has
// nothing to trigger.
func (s *Service) watchedBoards(ctx context.Context) ([]int, error) {
	wfs, err := s.Workflows.List(ctx)
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

// conditions builds the ConnectWise filter: changed after from, on one of boards. Closed tickets
// are included, because a missed closing update matters to status triggers.
func conditions(from time.Time, boards []int) string {
	parts := make([]string, len(boards))
	for i, id := range boards {
		parts[i] = fmt.Sprintf("board/id = %d", id)
	}
	return fmt.Sprintf("lastUpdated > [%s] AND (%s)", from.UTC().Format("2006-01-02T15:04:05Z"), strings.Join(parts, " OR "))
}

func storedOrNil(t time.Time, ok bool) any {
	if !ok {
		return nil
	}
	return t
}
