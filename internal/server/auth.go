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
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
	"golang.org/x/crypto/bcrypt"
)

type BootstrapResult struct {
	FilePath, Key string
}

type APIKeyPayload struct {
	EmailAddress string `json:"email_address"`
}

func (s *Server) handleCreateAPIKey(c *gin.Context) {
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

func (s *Server) BootstrapAdmin(ctx context.Context, dir string) (*BootstrapResult, error) {
	if dir == "" {
		return nil, errors.New("key directory cannot be blank")
	}

	if _, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("path %s is not a directory", dir)
	}

	slog.Debug("checking initial admin value")
	email := s.Config.General.InitialAdminEmail
	if email == "" {
		return nil, errors.New("initial admin config field must not be blank")
	}

	u, err := s.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("initial admin not found in db - creating now", "email", email)
			u, err = s.Queries.InsertUser(ctx, email)
			if err != nil {
				return nil, fmt.Errorf("creating admin user: %w", err)
			}
		} else {
			return nil, err
		}
	} else {
		slog.Debug("initial admin found in db", "email", email)
	}

	keys, err := s.Queries.ListAPIKeys(ctx)
	if err != nil {
		return nil, err
	}

	hasKey := false
	for _, k := range keys {
		if k.UserID == int(u.ID) {
			hasKey = true
			break
		}
	}

	if hasKey {
		return nil, errors.New("bootstrap user/key already exists")
	}

	key, err := s.createAPIKey(ctx, int(u.ID))
	if err != nil {
		return nil, fmt.Errorf("creating bootstrap key: %w", err)
	}

	path := filepath.Join(dir, "bootstrap.key")
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return nil, fmt.Errorf("writing key file: %w", err)
	}

	defer f.Close()

	if _, err := f.WriteString(key); err != nil {
		return nil, fmt.Errorf("writing key file: %w", err)
	}
	slog.Debug("bootstrap key saved")

	r := &BootstrapResult{
		FilePath: path,
		Key:      key,
	}
	return r, nil
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
