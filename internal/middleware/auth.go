package middleware

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "tb_session"

// SSOAuth is the slice of entra.Auth the middleware needs: a fallback that admits a request
// carrying a live Microsoft session and places the user in the request context.
type SSOAuth interface {
	RequireAuth(next http.Handler) http.Handler
}

// CombinedAuth accepts a session cookie, a Bearer API key or, when sso is non-nil, an Entra
// session, and stores the resolved user in the request context for User and UserID.
// userFromSSO extracts the user entra placed in the context; it is injected so this package
// does not depend on the generic entra.UserFromContext instantiation.
func CombinedAuth(keys repos.APIKeyRepository, users repos.APIUserRepository, auth *authsvc.Service, sso SSOAuth, userFromSSO func(context.Context) (*models.APIUser, bool)) Middleware {
	return func(next http.Handler) http.Handler {
		admit := func(w http.ResponseWriter, r *http.Request, id int) bool {
			u, err := users.Get(r.Context(), id)
			if err != nil {
				return false
			}
			next.ServeHTTP(w, r.WithContext(withUser(r.Context(), u)))
			return true
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try session cookie first
			if ck, err := r.Cookie(sessionCookie); err == nil && ck.Value != "" {
				if userID, err := auth.ValidateToken(r.Context(), ck.Value); err == nil && admit(w, r, userID) {
					return
				}
			}

			// Fall back to Bearer API key
			if key, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok && key != "" {
				allKeys, err := keys.List(r.Context())
				if err != nil {
					writeError(w, http.StatusInternalServerError, "db error")
					return
				}
				for _, k := range allKeys {
					if bcrypt.CompareHashAndPassword(k.KeyHash, []byte(key)) == nil {
						slog.Info("authenticated via api key", "user_id", k.UserID)
						if admit(w, r, k.UserID) {
							return
						}
						break
					}
				}
			}

			// Finally an Entra session. entra's Unauthorized hook writes the 401 when there is none.
			if sso != nil {
				sso.RequireAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					u, ok := userFromSSO(r.Context())
					if !ok {
						writeError(w, http.StatusUnauthorized, "authentication required")
						return
					}
					next.ServeHTTP(w, r.WithContext(withUser(r.Context(), u)))
				})).ServeHTTP(w, r)
				return
			}

			writeError(w, http.StatusUnauthorized, "authentication required")
		})
	}
}

// RequireRole admits only users whose role grants at least min. It must run inside CombinedAuth.
func RequireRole(min models.Role) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u := User(r.Context())
			if u == nil {
				writeError(w, http.StatusUnauthorized, "authentication required")
				return
			}
			if !u.Role.AtLeast(min) {
				writeError(w, http.StatusForbidden, "forbidden")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WriteUnauthorized is the JSON 401 the auth middleware emits, exported so the entra
// configuration can answer unauthenticated SSO fallbacks the same way.
func WriteUnauthorized(w http.ResponseWriter, _ *http.Request) {
	writeError(w, http.StatusUnauthorized, "authentication required")
}
