package intake

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Business hours for the staleness check: ConnectWise is quiet outside them, so silence then is
// not a signal.
var (
	businessZone     = mustZone("America/Chicago")
	businessOpenMin  = 7*60 + 30 // 07:30
	businessCloseMin = 19 * 60   // 19:00
	staleCheckEvery  = time.Minute
)

func mustZone(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.UTC
	}
	return loc
}

// inBusinessHours reports whether t falls on a weekday between open and close, Central time.
func inBusinessHours(t time.Time) bool {
	t = t.In(businessZone)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return false
	}
	m := t.Hour()*60 + t.Minute()
	return m >= businessOpenMin && m < businessCloseMin
}

// staleWatch alerts when no webhook has arrived for the configured number of business-hours
// minutes, and again when they resume. It is a method so it shares the repo, config and alerter.
type staleWatch struct {
	svc     *Service
	started time.Time
	alerted bool
}

func (s *Service) runStaleWatch(ctx context.Context) {
	defer s.wg.Done()
	w := &staleWatch{svc: s, started: s.now()}
	ticker := time.NewTicker(staleCheckEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.check(ctx)
		case <-ctx.Done():
			return
		}
	}
}

// check runs one evaluation. Silence outside business hours never alerts; an alert raised during
// the day clears as soon as a webhook arrives, whatever the hour.
func (w *staleWatch) check(ctx context.Context) {
	s := w.svc
	minutes := 0
	if s.Cfg != nil {
		minutes = s.Cfg.GetStaleAlertMinutes()
	}
	if minutes <= 0 {
		w.alerted = false
		return
	}

	st, err := s.Repo.Stats(ctx)
	if err != nil {
		if ctx.Err() == nil {
			slog.Warn("intake: staleness check could not read the queue", "error", err.Error())
		}
		return
	}
	last := w.started
	if st.LastReceivedAt != nil && st.LastReceivedAt.After(last) {
		last = *st.LastReceivedAt
	}
	now := s.now()
	silence := now.Sub(last)
	threshold := time.Duration(minutes) * time.Minute

	switch {
	case w.alerted && silence < threshold:
		w.alerted = false
		s.Alerter.Alert(ctx, "Ticket webhooks resumed", fmt.Sprintf("A webhook arrived at %s.", last.In(businessZone).Format("3:04pm MST")))
	case !w.alerted && silence >= threshold && inBusinessHours(now):
		w.alerted = true
		s.Alerter.Alert(ctx, fmt.Sprintf("No ticket webhooks for %d minutes", int(silence.Minutes())),
			fmt.Sprintf("Last one arrived %s. Check that the app is reachable from ConnectWise and that its callback is still registered.", last.In(businessZone).Format("Mon 3:04pm MST")))
	}
}
