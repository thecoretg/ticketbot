package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type APIKeyPayload struct {
	EmailAddress string `json:"email_address"`
}

func (s *Server) addKeysGroup() {
	keys := s.GinEngine.Group("/keys", s.APIKeyAuth(), ErrorHandler(s.Config.General.ExitOnError))
	keys.POST("/", s.CreateAPIKeyHandler)
}
func (s *Server) APIKeyAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		key := strings.TrimPrefix(auth, "Bearer ")

		keys, err := s.Queries.ListAPIKeys(c.Request.Context())
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "db error"})
			return
		}

		var userID int32
		found := false
		for _, k := range keys {
			if bcrypt.CompareHashAndPassword(k.KeyHash, []byte(key)) == nil {
				userID = int32(k.UserID)
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

func (s *Server) CreateAPIKeyHandler(c *gin.Context) {
	p := &APIKeyPayload{}

	if err := c.ShouldBindJSON(p); err != nil {
		c.Error(fmt.Errorf("unmarshaling payload: %w", err))
		return
	}

	u, err := s.Queries.GetUserByEmail(c.Request.Context(), p.EmailAddress)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Error(fmt.Errorf("querying user by email: %w", err))
		return
	}

	key, err := s.createAPIKey(c.Request.Context(), int(u.ID))
	if err != nil {
		c.Error(fmt.Errorf("creating API key: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": key})
}

func (s *Server) BootstrapAdmin(ctx context.Context) error {
	slog.Info("checking initial admin value")
	email := s.Config.General.InitialAdminEmail
	if email == "" {
		return errors.New("initial admin config field must not be blank")
	}

	u, err := s.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Info("initial admin not found in db - creating now", "email", email)
			u, err = s.Queries.InsertUser(ctx, email)
			if err != nil {
				return fmt.Errorf("creating admin user: %w", err)
			}
		} else {
			return err
		}
	} else {
		slog.Info("initial admin found in db", "email", email)
	}

	keys, err := s.Queries.ListAPIKeys(ctx)
	if err != nil {
		return err
	}

	hasKey := false
	for _, k := range keys {
		if k.UserID == int(u.ID) {
			hasKey = true
			break
		}
	}

	if !hasKey {
		key, err := s.createAPIKey(ctx, int(u.ID))
		if err != nil {
			return fmt.Errorf("creating bootstrap key: %w", err)
		}
		path := "boostrap.key"
		if err := os.WriteFile(path, []byte(key), 0600); err != nil {
			return fmt.Errorf("writing key file: %w", err)
		}
		slog.Debug("bootstrap key saved")
	}

	return nil
}

func (s *Server) createAPIKey(ctx context.Context, userID int) (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating key: %w", err)
	}

	plain := base64.RawURLEncoding.EncodeToString(raw)

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing key: %w", err)
	}

	params := db.InsertAPIKeyParams{
		UserID:  userID,
		KeyHash: hash,
	}
	_, err = s.Queries.InsertAPIKey(ctx, params)
	if err != nil {
		return "", fmt.Errorf("storing key: %w", err)
	}

	return plain, nil
}
