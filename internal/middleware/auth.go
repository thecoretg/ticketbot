package middleware

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/repos"
	"golang.org/x/crypto/bcrypt"
)

func APIKeyAuth(r repos.APIKeyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")
		if key == "" {
			slog.Warn("auth middleware: got empty key in request header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "empty api key"})
			return
		}
		hash, err := hashKey(key)
		if err == nil {
			slog.Debug("auth middleware: got key from request", "hash", hash)
		}

		keys, err := r.List(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		if len(keys) > 0 {
			slog.Debug("auth middleware: got keys from store", "total", len(keys))
			for _, k := range keys {
				slog.Debug("auth middleware: key from store", "user_id", k.UserID, "hash", k.KeyHash)
			}
		} else {
			slog.Debug("auth middleware: got no keys from store")
		}

		var userID int
		found := false
		for _, k := range keys {
			if bcrypt.CompareHashAndPassword(k.KeyHash, []byte(key)) == nil {
				userID = k.UserID
				slog.Info("authenticated user", "user_id", userID)
				found = true
				break
			}
		}

		if !found {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}

func hashKey(key string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
}
