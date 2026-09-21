package intake

import (
	"context"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

type staleCfg int

func (c staleCfg) GetLogRetentionDays() int  { return 7 }
func (c staleCfg) GetStaleAlertMinutes() int { return int(c) }

type staleRepo struct {
	fakeRepo
	last *time.Time
}

func (r *staleRepo) Stats(context.Context) (*models.IntakeStats, error) {
	return &models.IntakeStats{LastReceivedAt: r.last}, nil
}

// tuesday10 is a Tuesday at 10:00 Central, inside business hours.
var tuesday10 = time.Date(2026, 9, 22, 15, 0, 0, 0, time.UTC)

func TestStaleWatchAlertsOnceInBusinessHoursAndRecovers(t *testing.T) {
	repo, alerter := &staleRepo{}, &fakeAlerter{}
	s := New(Params{Repo: repo, Processor: &fakeProcessor{}, Alerter: alerter, Cfg: staleCfg(60)})
	now := tuesday10
	s.now = func() time.Time { return now }
	w := &staleWatch{svc: s, started: now}

	now = tuesday10.Add(30 * time.Minute)
	w.check(context.Background())
	if len(alerter.subjects) != 0 {
		t.Fatal("30 minutes of silence is under the threshold")
	}

	now = tuesday10.Add(61 * time.Minute)
	w.check(context.Background())
	w.check(context.Background())
	if len(alerter.subjects) != 1 {
		t.Fatalf("alerts = %v, want exactly one", alerter.subjects)
	}

	arrived := tuesday10.Add(70 * time.Minute)
	repo.last = &arrived
	now = tuesday10.Add(71 * time.Minute)
	w.check(context.Background())
	if len(alerter.subjects) != 2 || alerter.subjects[1] != "Ticket webhooks resumed" {
		t.Fatalf("alerts = %v, want a recovery", alerter.subjects)
	}
}

func TestStaleWatchIgnoresSilenceOutsideBusinessHours(t *testing.T) {
	repo, alerter := &staleRepo{}, &fakeAlerter{}
	s := New(Params{Repo: repo, Processor: &fakeProcessor{}, Alerter: alerter, Cfg: staleCfg(60)})
	saturday := time.Date(2026, 9, 26, 15, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 9, 22, 2, 0, 0, 0, time.UTC) // Monday 9pm Central
	for _, at := range []time.Time{saturday, evening} {
		now := at
		s.now = func() time.Time { return now }
		w := &staleWatch{svc: s, started: at.Add(-3 * time.Hour)}
		w.check(context.Background())
	}
	if len(alerter.subjects) != 0 {
		t.Fatalf("alerts = %v, want none outside business hours", alerter.subjects)
	}
}

func TestStaleWatchDisabledAtZero(t *testing.T) {
	repo, alerter := &staleRepo{}, &fakeAlerter{}
	s := New(Params{Repo: repo, Processor: &fakeProcessor{}, Alerter: alerter, Cfg: staleCfg(0)})
	s.now = func() time.Time { return tuesday10.Add(5 * time.Hour) }
	w := &staleWatch{svc: s, started: tuesday10}
	w.check(context.Background())
	if len(alerter.subjects) != 0 {
		t.Fatal("0 disables the check")
	}
}

func TestInBusinessHours(t *testing.T) {
	cases := map[time.Time]bool{
		tuesday10: true,
		time.Date(2026, 9, 22, 12, 29, 0, 0, time.UTC): false, // 07:29 Central
		time.Date(2026, 9, 22, 12, 30, 0, 0, time.UTC): true,  // 07:30 Central
		time.Date(2026, 9, 22, 23, 59, 0, 0, time.UTC): true,  // 18:59 Central
		time.Date(2026, 9, 23, 0, 0, 0, 0, time.UTC):   false, // 19:00 Central
		time.Date(2026, 9, 26, 15, 0, 0, 0, time.UTC):  false, // Saturday
	}
	for at, want := range cases {
		if got := inBusinessHours(at); got != want {
			t.Errorf("inBusinessHours(%s) = %v, want %v", at.In(businessZone), got, want)
		}
	}
}
