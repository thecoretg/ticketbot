package catchup

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

var now = time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC)

type fakeCW struct {
	tix    []psa.Ticket
	err    error
	params []map[string]string
}

func (f *fakeCW) ListTickets(_ context.Context, params map[string]string, _ ...psa.ListOption) ([]psa.Ticket, error) {
	f.params = append(f.params, params)
	return f.tix, f.err
}

type fakeQueue struct{ ids []int }

func (f *fakeQueue) Enqueue(_ context.Context, id int, action models.IntakeAction, _ []byte) (*models.WebhookIntake, error) {
	if action != models.IntakeCatchup {
		return nil, errors.New("wrong action " + string(action))
	}
	f.ids = append(f.ids, id)
	return &models.WebhookIntake{TicketID: id, Action: action}, nil
}

type fakeOpen []int

func (f fakeOpen) OpenTickets(context.Context, []int) ([]int, error) { return f, nil }

type fakeStored map[int]time.Time

func (f fakeStored) CWLastUpdated(context.Context, []int) (map[int]time.Time, error) { return f, nil }

type fakeWorkflows []*models.Workflow

func (f fakeWorkflows) List(context.Context) ([]*models.Workflow, error) { return f, nil }

type fakeState struct{ saved []models.CatchupState }

func (f *fakeState) Get(context.Context) (*models.CatchupState, error) {
	if len(f.saved) == 0 {
		return nil, nil
	}
	st := f.saved[len(f.saved)-1]
	return &st, nil
}

func (f *fakeState) Save(_ context.Context, st *models.CatchupState) error {
	f.saved = append(f.saved, *st)
	return nil
}

type minutes int

func (m minutes) GetCatchupIntervalMinutes() int { return int(m) }

func ticket(id int, updated, entered time.Time) psa.Ticket {
	var t psa.Ticket
	t.ID = id
	t.Info.LastUpdated = updated
	t.Info.DateEntered = entered
	return t
}

func newSvc(cw *fakeCW, q *fakeQueue, open fakeOpen, stored fakeStored, st *fakeState) *Service {
	s := New(Params{CW: cw, Queue: q, Intake: open, Tickets: stored, State: st, Cfg: minutes(15),
		Workflows: fakeWorkflows{{BoardID: 7, Enabled: true}, {BoardID: 3, Enabled: true}, {BoardID: 9, Enabled: false}}})
	s.now = func() time.Time { return now }
	return s
}

func TestRunQueuesOnlyMissedTickets(t *testing.T) {
	old := now.Add(-48 * time.Hour)
	cw := &fakeCW{tix: []psa.Ticket{
		ticket(1, now.Add(-2*time.Minute), old),                     // stored copy is older: webhook missed
		ticket(2, now.Add(-3*time.Minute), old),                     // stored copy matches: webhook arrived
		ticket(3, now.Add(-4*time.Minute), old),                     // older stored copy, but a row is already queued
		ticket(4, now.Add(-5*time.Minute), now.Add(-6*time.Minute)), // new ticket whose added webhook was missed
		ticket(5, now.Add(-5*time.Minute), old),                     // never stored and not new: skipped
	}}
	stored := fakeStored{
		1: now.Add(-time.Hour),
		2: now.Add(-3 * time.Minute),
		3: now.Add(-time.Hour),
	}
	q, st := &fakeQueue{}, &fakeState{}
	n, err := newSvc(cw, q, fakeOpen{3}, stored, st).Run(context.Background(), nil, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if want := []int{1, 4}; !reflect.DeepEqual(q.ids, want) || n != 2 {
		t.Errorf("queued %v (n=%d), want %v", q.ids, n, want)
	}
	got := st.saved[0]
	if got.CheckedThrough == nil || !got.CheckedThrough.Equal(now) || got.LastQueued != 2 || got.LastError != nil {
		t.Errorf("state = %+v, want watermark at the run start and 2 queued", got)
	}
}

func TestRunWindowAndConditions(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state *models.CatchupState
		from  string
	}{
		{"first run looks back one interval", nil, "2026-09-29T14:40:00Z"},
		{"later runs start at the watermark", &models.CatchupState{CheckedThrough: new(now.Add(-40 * time.Minute))}, "2026-09-29T14:15:00Z"},
		{"long downtime is capped at a day", &models.CatchupState{CheckedThrough: new(now.Add(-72 * time.Hour))}, "2026-09-28T14:55:00Z"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cw := &fakeCW{}
			if _, err := newSvc(cw, &fakeQueue{}, nil, fakeStored{}, &fakeState{}).Run(context.Background(), tc.state, 15*time.Minute); err != nil {
				t.Fatal(err)
			}
			want := "lastUpdated > [" + tc.from + "] AND (board/id = 3 OR board/id = 7)"
			if got := cw.params[0]["conditions"]; got != want {
				t.Errorf("conditions = %q, want %q", got, want)
			}
			if f := cw.params[0]["fields"]; !strings.Contains(f, "_info/lastUpdated") {
				t.Errorf("fields = %q, want only the ids and timestamps", f)
			}
		})
	}
}

func TestRunFailureKeepsWatermark(t *testing.T) {
	mark := now.Add(-20 * time.Minute)
	cw := &fakeCW{err: errors.New("connectwise down")}
	st := &fakeState{}
	_, err := newSvc(cw, &fakeQueue{}, nil, fakeStored{}, st).Run(context.Background(), &models.CatchupState{CheckedThrough: &mark}, 15*time.Minute)
	if err == nil {
		t.Fatal("want the connectwise error")
	}
	got := st.saved[0]
	if got.CheckedThrough == nil || !got.CheckedThrough.Equal(mark) {
		t.Errorf("watermark moved to %v, want it kept at %v", got.CheckedThrough, mark)
	}
	if got.LastRunAt == nil || !got.LastRunAt.Equal(now) || got.LastError == nil {
		t.Errorf("state = %+v, want the failed run and its error recorded", got)
	}
}

func TestRunWithoutEnabledWorkflowsSkipsConnectWise(t *testing.T) {
	cw := &fakeCW{}
	s := newSvc(cw, &fakeQueue{}, nil, fakeStored{}, &fakeState{})
	s.Workflows = fakeWorkflows{{BoardID: 9, Enabled: false}}
	if _, err := s.Run(context.Background(), nil, 15*time.Minute); err != nil {
		t.Fatal(err)
	}
	if len(cw.params) != 0 {
		t.Errorf("made %d connectwise calls, want none", len(cw.params))
	}
}

func TestTickHonoursInterval(t *testing.T) {
	for _, tc := range []struct {
		name    string
		cfg     minutes
		lastRun time.Duration // before now; 0 means never
		wantRun bool
	}{
		{"never run", 15, 0, true},
		{"interval not yet passed", 15, 10 * time.Minute, false},
		{"interval passed", 15, 15 * time.Minute, true},
		{"switched off", 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cw, st := &fakeCW{}, &fakeState{}
			if tc.lastRun > 0 {
				st.saved = []models.CatchupState{{LastRunAt: new(now.Add(-tc.lastRun)), CheckedThrough: new(now.Add(-tc.lastRun))}}
			}
			s := newSvc(cw, &fakeQueue{}, nil, fakeStored{}, st)
			s.Cfg = tc.cfg
			s.tick(context.Background())
			if ran := len(cw.params) == 1; ran != tc.wantRun {
				t.Errorf("ran = %v, want %v", ran, tc.wantRun)
			}
		})
	}
}
