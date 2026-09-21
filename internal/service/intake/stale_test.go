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
func (c staleCfg) BusinessWindow() models.BusinessWindow {
	return (&models.Config{}).BusinessWindow() // the defaults: 07:30 to 19:00 Central, weekdays
}

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

func TestStaleWatchHonoursAConfiguredWindow(t *testing.T) {
	// Saturday 10:00 Central is silent by default; a window that includes Saturday alerts.
	repo, alerter := &staleRepo{}, &fakeAlerter{}
	saturday := time.Date(2026, 9, 26, 15, 0, 0, 0, time.UTC)
	cfg := &models.Config{StaleAlertMinutes: 60, BusinessOpen: "08:00", BusinessClose: "12:00", BusinessDays: "sat", BusinessZone: "America/Chicago"}
	s := New(Params{Repo: repo, Processor: &fakeProcessor{}, Alerter: alerter, Cfg: cfg})
	s.now = func() time.Time { return saturday }
	w := &staleWatch{svc: s, started: saturday.Add(-3 * time.Hour)}
	w.check(context.Background())
	if len(alerter.subjects) != 1 {
		t.Fatalf("alerts = %v, want one inside the configured Saturday window", alerter.subjects)
	}
}
