package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) createAPIKey(ctx context.Context, email string, explicitKey *string) (string, error) {
	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			return "", err
		}
		return "", fmt.Errorf("getting user by email: %w", err)
	}

	plain, err := generateKey()
	if err != nil {
		return "", err
	}

	if explicitKey != nil {
		plain = *explicitKey
	}

	hash, err := hashKey(plain)
	if err != nil {
		return "", fmt.Errorf("hashing key: %w", err)
	}

	hint := generateKeyHint(plain)
	p := &models.APIKey{
		UserID:  u.ID,
		KeyHash: hash,
		KeyHint: &hint,
	}

	_, err = s.Keys.Insert(ctx, p)
	if err != nil {
		return "", fmt.Errorf("storing key: %w", err)
	}

	return plain, nil
}

func (s *Service) BootstrapAdmin(ctx context.Context, email string, explicitKey *string) error {
	if explicitKey != nil {
		slog.Debug("bootstrapping admin with explicit key from test flags")
	}

	if email == "" {
		return errors.New("received empty email")
	}

	u, err := s.Users.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			slog.Info("initial admin not found; creating now", "email", email)
			u, err = s.Users.Insert(ctx, email)
			if err != nil {
				return fmt.Errorf("creating user: %w", err)
			}
		} else {
			return fmt.Errorf("getting admin by email: %w", err)
		}
	} else {
		slog.Info("initial admin found in store")
	}

	keys, err := s.Keys.List(ctx)
	if err != nil {
		return fmt.Errorf("getting keys: %w", err)
	}

	hasKey := false
	for _, k := range keys {
		if k.UserID == u.ID {
			hasKey = true
			break
		}
	}

	if hasKey {
		slog.Info("initial admin already has an api token, skipping creation")
		return nil
	}

	key, err := s.createAPIKey(ctx, email, explicitKey)
	if err != nil {
		return fmt.Errorf("creating key: %w", err)
	}

	slog.Info("bootstrap token created", "email", email, "key", key)
	if os.Getenv("API_KEY_DELAY") == "true" {
		slog.Info("waiting 30 seconds; copy the above key, it won't be shown again")
		time.Sleep(30 * time.Second)
	}

	return nil
}

func generateKey() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating key: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func hashKey(key string) ([]byte, error) {
	return bcrypt.GenerateFromPassword([]byte(key), bcrypt.DefaultCost)
}

func generateKeyHint(key string) string {
	if len(key) <= 4 {
		return key
	}
	return key[len(key)-4:]
}
