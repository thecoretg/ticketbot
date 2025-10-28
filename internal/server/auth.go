package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

func (cl *Client) handleCreateAPIKey(c *gin.Context) {
	p := &APIKeyPayload{}

	if err := c.ShouldBindJSON(p); err != nil {
		c.Error(fmt.Errorf("unmarshaling payload: %w", err))
		return
	}

	u, err := cl.Queries.GetUserByEmail(c.Request.Context(), p.EmailAddress)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.Status(http.StatusNotFound)
			return
		}
		c.Error(fmt.Errorf("querying user by email: %w", err))
		return
	}

	key, err := cl.createAPIKey(c.Request.Context(), u.ID)
	if err != nil {
		c.Error(fmt.Errorf("creating API key: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"api_key": key})
}

func (cl *Client) bootstrapAdmin(ctx context.Context) (string, error) {
	email := cl.Config.InitialAdminEmail
	if email == "" {
		return "", errors.New("initial admin config field must not be blank")
	}

	u, err := cl.Queries.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("initial admin not found in db - creating now", "email", email)
			u, err = cl.Queries.InsertUser(ctx, email)
			if err != nil {
				return "", fmt.Errorf("creating admin user: %w", err)
			}
		} else {
			return "", err
		}
	} else {
		slog.Debug("initial admin found in db", "email", email)
	}

	keys, err := cl.Queries.ListAPIKeys(ctx)
	if err != nil {
		return "", err
	}

	hasKey := false
	for _, k := range keys {
		if k.UserID == u.ID {
			hasKey = true
			break
		}
	}

	if hasKey {
		return "", nil
	}

	key, err := cl.createAPIKey(ctx, u.ID)
	if err != nil {
		return "", fmt.Errorf("creating bootstrap key: %w", err)
	}

	return key, nil
}

func (cl *Client) createAPIKey(ctx context.Context, userID int) (string, error) {
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
	_, err = cl.Queries.InsertAPIKey(ctx, params)
	if err != nil {
		return "", fmt.Errorf("storing key: %w", err)
	}

	return plain, nil
}
