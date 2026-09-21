package ticketbot

import (
	"context"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
)

// HistoryPurger ages out workflow run summaries and ticket history events together, on the
// history retention setting. It is driven by the intake service's hourly purge loop.
type HistoryPurger struct {
	Runs   repos.WorkflowRunRepository
	Events repos.TicketEventRepository
	Cfg    interface{ GetHistoryRetentionDays() int }
}

func (h *HistoryPurger) Purge(ctx context.Context, now time.Time) {
	days := h.Cfg.GetHistoryRetentionDays()
	if days <= 0 {
		return
	}
	cutoff := now.AddDate(0, 0, -days)
	runs, err := h.Runs.DeleteBefore(ctx, cutoff)
	if err != nil {
		slog.Warn("history: purging run summaries", "error", err.Error())
		return
	}
	events, err := h.Events.DeleteBefore(ctx, cutoff)
	if err != nil {
		slog.Warn("history: purging ticket events", "error", err.Error())
		return
	}
	if runs > 0 || events > 0 {
		slog.Info("history: purged", "runs", runs, "events", events, "retention_days", days)
	}
}
