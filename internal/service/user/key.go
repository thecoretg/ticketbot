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

func (s *Service) BootstrapAdmin(ctx context.Context, email string, explicitKey *string, explicitPassword *string) error {
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

	if !hasKey {
		key, err := s.createAPIKey(ctx, email, explicitKey)
		if err != nil {
			return fmt.Errorf("creating key: %w", err)
		}

		slog.Info("bootstrap token created", "email", email, "key", key)
		if os.Getenv("API_KEY_DELAY") == "true" {
			slog.Info("waiting 30 seconds; copy the above key, it won't be shown again")
			time.Sleep(30 * time.Second)
		}
	} else {
		slog.Info("initial admin already has an api token, skipping creation")
	}

	if err := s.bootstrapPassword(ctx, u.ID, email, explicitPassword); err != nil {
		return fmt.Errorf("bootstrapping admin password: %w", err)
	}

	return nil
}

func (s *Service) bootstrapPassword(ctx context.Context, userID int, email string, explicitPassword *string) error {
	ua, err := s.Users.GetForAuth(ctx, email)
	if err != nil {
		return fmt.Errorf("checking user auth record: %w", err)
	}

	if len(ua.PasswordHash) > 0 {
		slog.Info("initial admin already has a password, skipping")
		return nil
	}

	// Determine the password to set
	var plain string
	if explicitPassword != nil && *explicitPassword != "" {
		plain = *explicitPassword
	} else {
		generated, err := generatePassword()
		if err != nil {
			return fmt.Errorf("generating password: %w", err)
		}
		plain = generated
		slog.Info("bootstrap password generated (copy now, won't be shown again)", "password", plain)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.Users.SetPassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("storing password: %w", err)
	}

	return nil
}

func generatePassword() (string, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generating password: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(raw), nil
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
