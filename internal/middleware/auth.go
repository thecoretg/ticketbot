package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "tb_session"

// CombinedAuth accepts either a valid session cookie or a Bearer API key and stores the user ID in
// the request context for UserID.
func CombinedAuth(keys repos.APIKeyRepository, auth *authsvc.Service) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Try session cookie first
			if ck, err := r.Cookie(sessionCookie); err == nil && ck.Value != "" {
				if userID, err := auth.ValidateToken(r.Context(), ck.Value); err == nil {
					next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), userID)))
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
						next.ServeHTTP(w, r.WithContext(withUserID(r.Context(), k.UserID)))
						return
					}
				}
			}

			writeError(w, http.StatusUnauthorized, "authentication required")
		})
	}
}
