package authsvc

import (
	"context"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
)

// SessionPurger implements intake.Purger for everything sign-in leaves behind: dashboard
// sessions, half-finished TOTP sign-ins and Entra sessions and flow states. Each expires in the
// query that reads it but was never deleted.
type SessionPurger struct {
	Sessions    repos.SessionRepository
	TOTPPending repos.TOTPPendingRepository
	SSO         repos.SSOStore
}

func (p *SessionPurger) Purge(ctx context.Context, _ time.Time) {
	if err := p.Sessions.DeleteExpired(ctx); err != nil {
		slog.Warn("auth: purging expired sessions", "error", err)
	}
	if err := p.TOTPPending.DeleteExpired(ctx); err != nil {
		slog.Warn("auth: purging expired totp sign-ins", "error", err)
	}
	if err := p.SSO.DeleteExpired(ctx); err != nil {
		slog.Warn("auth: purging expired sso rows", "error", err)
	}
}
