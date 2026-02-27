package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const sessionCookie = "tb_session"

// CombinedAuth accepts either a Bearer API key or a valid session cookie.
func CombinedAuth(keys repos.APIKeyRepository, auth *authsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Try session cookie first
		if token, err := c.Cookie(sessionCookie); err == nil && token != "" {
			userID, err := auth.ValidateToken(c.Request.Context(), token)
			if err == nil {
				c.Set("user_id", userID)
				c.Next()
				return
			}
		}

		// Fall back to Bearer API key
		authHeader := c.GetHeader("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			key := strings.TrimPrefix(authHeader, "Bearer ")
			if key != "" {
				allKeys, err := keys.List(c.Request.Context())
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
					return
				}

				for _, k := range allKeys {
					if bcrypt.CompareHashAndPassword(k.KeyHash, []byte(key)) == nil {
						slog.Info("authenticated via api key", "user_id", k.UserID)
						c.Set("user_id", k.UserID)
						c.Next()
						return
					}
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
	}
}

// SessionAuth accepts only a valid session cookie (used for panel-only routes if needed).
func SessionAuth(auth *authsvc.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(sessionCookie)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		userID, err := auth.ValidateToken(c.Request.Context(), token)
		if err != nil {
			if isNotFound(err) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session expired or invalid"})
				return
			}
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "session validation failed"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func isNotFound(err error) bool {
	return err != nil && err.Error() == models.ErrSessionNotFound.Error()
}
