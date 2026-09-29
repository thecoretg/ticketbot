package syncsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

type fakeSyncer struct {
	busy     bool
	err      error
	payloads []*models.SyncPayload
	by       []string
}

func (f *fakeSyncer) IsSyncing() bool { return f.busy }

func (f *fakeSyncer) RunAndWait(_ context.Context, p *models.SyncPayload, by string) error {
	f.payloads = append(f.payloads, p)
	f.by = append(f.by, by)
	return f.err
}

type fakeCallbacks struct {
	added bool
	err   error
	calls int
}

func (f *fakeCallbacks) EnsureTicketCallback(context.Context) (bool, error) {
	f.calls++
	return f.added, f.err
}

type fakeWorkflowList []*models.Workflow

func (f fakeWorkflowList) List(context.Context) ([]*models.Workflow, error) { return f, nil }

type fakeJobs struct{ saved []models.ScheduledJob }

func (f *fakeJobs) Get(context.Context, string) (*models.ScheduledJob, error) {
	if len(f.saved) == 0 {
		return nil, nil
	}
	j := f.saved[len(f.saved)-1]
	return &j, nil
}

func (f *fakeJobs) Save(_ context.Context, j *models.ScheduledJob) error {
	f.saved = append(f.saved, *j)
	return nil
}

type fakeAlerts struct{ subjects []string }

func (f *fakeAlerts) Alert(_ context.Context, subject, _ string) {
	f.subjects = append(f.subjects, subject)
}

var chicago, _ = time.LoadLocation("America/Chicago")

func nightlyCfg() *models.Config {
	cfg := models.DefaultConfig
	return &cfg
}

func newNightly(sync *fakeSyncer, cb *fakeCallbacks, jobs *fakeJobs, al *fakeAlerts, now time.Time) *Nightly {
	n := NewNightly(Nightly{
		Sync: sync, Jobs: jobs, Cfg: nightlyCfg(), Alerter: al,
		Workflows: fakeWorkflowList{{BoardID: 34, Enabled: true}, {BoardID: 12, Enabled: true}, {BoardID: 7}},
	})
	if cb != nil {
		n.Callbacks = cb
	}
	n.now = func() time.Time { return now }
	return n
}

func TestNightlyDue(t *testing.T) {
	at := time.Date(2026, 9, 30, 2, 0, 0, 0, chicago)
	for _, tc := range []struct {
		name    string
		now     time.Time
		last    *time.Time
		enabled bool
		want    bool
	}{
		{"before the time", at.Add(-time.Minute), nil, true, false},
		{"at the time", at, nil, true, true},
		{"missed, back inside the window", at.Add(2 * time.Hour), nil, true, true},
		{"missed, back after the window", at.Add(4 * time.Hour), nil, true, false},
		{"already ran today", at.Add(10 * time.Minute), new(at.Add(time.Minute)), true, false},
		{"ran yesterday", at.Add(time.Minute), new(at.Add(-24 * time.Hour)), true, true},
		{"switched off", at, nil, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := newNightly(&fakeSyncer{}, nil, &fakeJobs{}, &fakeAlerts{}, tc.now)
			n.Cfg.NightlySyncEnabled = tc.enabled
			var job *models.ScheduledJob
			if tc.last != nil {
				job = &models.ScheduledJob{LastStartedAt: tc.last}
			}
			if _, got := n.due(tc.now, job); got != tc.want {
				t.Errorf("due = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNightlyRunSyncsEverythingOnWatchedBoards(t *testing.T) {
	sync, jobs, al := &fakeSyncer{}, &fakeJobs{}, &fakeAlerts{}
	now := time.Date(2026, 9, 30, 2, 0, 0, 0, chicago)
	n := newNightly(sync, &fakeCallbacks{}, jobs, al, now)
	n.Cfg.NightlySyncRunWorkflows = true
	n.Run(context.Background())

	if len(sync.payloads) != 1 {
		t.Fatalf("ran %d syncs, want 1", len(sync.payloads))
	}
	p := sync.payloads[0]
	if all := p.CWBoards && p.WebexRecipients && p.CWMembers && p.CWCompanies && p.CWContacts && p.CWTickets && p.RunWorkflows; !all {
		t.Errorf("payload = %+v, want every phase and workflows on", p)
	}
	if len(p.BoardIDs) != 2 || p.BoardIDs[0] != 12 || p.BoardIDs[1] != 34 {
		t.Errorf("board ids = %v, want the enabled workflows' boards [12 34]", p.BoardIDs)
	}
	if sync.by[0] != NightlyStartedBy {
		t.Errorf("started by %q", sync.by[0])
	}
	last := jobs.saved[len(jobs.saved)-1]
	if last.LastStartedAt == nil || last.LastFinishedAt == nil || last.LastError != nil {
		t.Errorf("record = %+v, want start and finish and no error", last)
	}
	if len(al.subjects) != 0 {
		t.Errorf("alerts = %v, want none", al.subjects)
	}
}

func TestNightlyRunWithoutWorkflowsSkipsTickets(t *testing.T) {
	sync := &fakeSyncer{}
	n := newNightly(sync, nil, &fakeJobs{}, &fakeAlerts{}, time.Now())
	n.Workflows = fakeWorkflowList{{BoardID: 7}}
	n.Run(context.Background())
	if p := sync.payloads[0]; p.CWTickets || p.RunWorkflows {
		t.Errorf("payload = %+v, want no ticket phase and workflows off by default", p)
	}
}

func TestNightlyRunAlerts(t *testing.T) {
	t.Run("lost callback", func(t *testing.T) {
		al := &fakeAlerts{}
		newNightly(&fakeSyncer{}, &fakeCallbacks{added: true}, &fakeJobs{}, al, time.Now()).Run(context.Background())
		if len(al.subjects) != 1 || !strings.Contains(al.subjects[0], "callback was missing") {
			t.Errorf("alerts = %v", al.subjects)
		}
	})
	t.Run("failed phases", func(t *testing.T) {
		al, jobs := &fakeAlerts{}, &fakeJobs{}
		sync := &fakeSyncer{err: errors.New("syncing connectwise members: listing connectwise members: 503")}
		newNightly(sync, &fakeCallbacks{}, jobs, al, time.Now()).Run(context.Background())
		if len(al.subjects) != 1 || !strings.Contains(al.subjects[0], "finished with errors") {
			t.Errorf("alerts = %v", al.subjects)
		}
		if last := jobs.saved[len(jobs.saved)-1]; last.LastError == nil || !strings.Contains(*last.LastError, "503") {
			t.Errorf("record = %+v, want the error kept", last)
		}
	})
}

func TestNightlyTickWaitsForARunningSync(t *testing.T) {
	sync, jobs := &fakeSyncer{busy: true}, &fakeJobs{}
	n := newNightly(sync, nil, jobs, &fakeAlerts{}, time.Date(2026, 9, 30, 2, 5, 0, 0, chicago))
	n.tick(context.Background())
	if len(sync.payloads) != 0 || len(jobs.saved) != 0 {
		t.Fatalf("ran while another sync was running")
	}
	sync.busy = false
	n.tick(context.Background())
	if len(sync.payloads) != 1 {
		t.Fatalf("did not run once the other sync finished")
	}
	n.tick(context.Background())
	if len(sync.payloads) != 1 {
		t.Errorf("ran twice in one night")
	}
}
