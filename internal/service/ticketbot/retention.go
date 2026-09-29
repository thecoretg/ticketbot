package ticketbot

import (
	"context"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
)

// TicketPurger hard-deletes tickets on the intake service's hourly purge loop. Soft-deleted
// tickets go once their delete is older than the history retention, when their history has aged
// out anyway. Closed tickets go only while closed-ticket retention is on. Both cascade to notes,
// notifications, history events and run summaries.
type TicketPurger struct {
	Tickets repos.TicketRepository
	Cfg     interface {
		GetHistoryRetentionDays() int
		ClosedTicketRetention() int
	}
}

func (p *TicketPurger) Purge(ctx context.Context, _ time.Time) {
	if days := p.Cfg.GetHistoryRetentionDays(); days > 0 {
		n, err := p.Tickets.DeleteSoftDeletedOlderThan(ctx, days)
		if err != nil {
			slog.Warn("tickets: purging soft-deleted tickets", "error", err.Error())
		} else if n > 0 {
			slog.Info("tickets: purged soft-deleted tickets", "deleted", n, "retention_days", days)
		}
	}
	if days := p.Cfg.ClosedTicketRetention(); days > 0 {
		n, err := p.Tickets.DeleteClosedOlderThan(ctx, days)
		if err != nil {
			slog.Warn("tickets: purging closed tickets", "error", err.Error())
		} else if n > 0 {
			slog.Info("tickets: purged closed tickets", "deleted", n, "retention_days", days)
		}
	}
}
