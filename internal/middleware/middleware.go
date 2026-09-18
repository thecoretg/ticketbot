// Package middleware holds the net/http middleware the server composes around its handlers.
package middleware

import (
	"context"
	"net/http"

	"github.com/thecoretg/ticketbot/models"
)

// Middleware wraps a handler.
type Middleware func(http.Handler) http.Handler

// Chain applies mws to h so that the first middleware is the outermost.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

type ctxKey int

const userKey ctxKey = iota

func withUser(ctx context.Context, u *models.APIUser) context.Context {
	return context.WithValue(ctx, userKey, u)
}

// User returns the authenticated user set by the auth middleware, or nil when unauthenticated.
func User(ctx context.Context) *models.APIUser {
	u, _ := ctx.Value(userKey).(*models.APIUser)
	return u
}

// UserID returns the authenticated user's ID, or 0 when unauthenticated.
func UserID(ctx context.Context) int {
	if u := User(ctx); u != nil {
		return u.ID
	}
	return 0
}

func writeError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"error":` + quote(msg) + `}`))
}

// quote JSON-escapes a short, controlled message without pulling in encoding/json.
func quote(s string) string {
	b := make([]byte, 0, len(s)+2)
	b = append(b, '"')
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\\':
			b = append(b, '\\', c)
		case '\n':
			b = append(b, '\\', 'n')
		default:
			b = append(b, c)
		}
	}
	return string(append(b, '"'))
}
