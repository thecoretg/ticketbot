package authsvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
)

// SessionPurger implements intake.Purger for dashboard sessions, which expire in the query
// that reads them but were never deleted.
type SessionPurger struct {
	Sessions repos.SessionRepository
}

func (p *SessionPurger) Purge(ctx context.Context, _ time.Time) {
	if err := p.Sessions.DeleteExpired(ctx); err != nil {
		slog.Warn("auth: purging expired sessions", "error", err)
	}
}
