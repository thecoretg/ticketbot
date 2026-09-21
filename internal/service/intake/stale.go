package intake

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// The business hours that bound the check come from app config (models.BusinessWindow):
// ConnectWise is quiet outside them, so silence then is not a signal.
const staleCheckEvery = time.Minute

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
	window := s.Cfg.BusinessWindow()
	loc := window.Loc
	if loc == nil {
		loc = time.UTC
	}

	switch {
	case w.alerted && silence < threshold:
		w.alerted = false
		s.Alerter.Alert(ctx, "Ticket webhooks resumed", fmt.Sprintf("A webhook arrived at %s.", last.In(loc).Format("3:04pm MST")))
	case !w.alerted && silence >= threshold && window.Contains(now):
		w.alerted = true
		s.Alerter.Alert(ctx, fmt.Sprintf("No ticket webhooks for %d minutes", int(silence.Minutes())),
			fmt.Sprintf("Last one arrived %s. Check that the app is reachable from ConnectWise and that its callback is still registered.", last.In(loc).Format("Mon 3:04pm MST")))
	}
}
