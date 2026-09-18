// Package middleware holds the net/http middleware the server composes around its handlers.
package middleware

import (
	"context"
	"net/http"
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

const userIDKey ctxKey = iota

func withUserID(ctx context.Context, id int) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID returns the authenticated user's ID set by the auth middleware, or 0 when unauthenticated.
func UserID(ctx context.Context) int {
	id, _ := ctx.Value(userIDKey).(int)
	return id
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
