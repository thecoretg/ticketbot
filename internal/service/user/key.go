package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const defaultBootstrapPassword = "password"

// BootstrapAdmin ensures the initial admin user exists. If the user does not
// exist yet, it is created and its password is set to initialPassword (or
// "password" if nil), with password_reset_required = true.
// If the user already exists, this is a no-op.
func (s *Service) BootstrapAdmin(ctx context.Context, email string, initialPassword *string) error {
	if email == "" {
		return errors.New("received empty email")
	}

	_, err := s.Users.GetByEmail(ctx, email)
	if err == nil {
		slog.Info("initial admin already exists, skipping bootstrap")
		return nil
	}
	if !errors.Is(err, models.ErrAPIUserNotFound) {
		return fmt.Errorf("getting admin by email: %w", err)
	}

	slog.Info("initial admin not found; creating now", "email", email)
	u, err := s.Users.Insert(ctx, email)
	if err != nil {
		return fmt.Errorf("creating user: %w", err)
	}

	plain := defaultBootstrapPassword
	if initialPassword != nil && *initialPassword != "" {
		plain = *initialPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	if err := s.Users.SetPassword(ctx, u.ID, hash); err != nil {
		return fmt.Errorf("setting password: %w", err)
	}

	if err := s.Users.SetPasswordResetRequired(ctx, u.ID, true); err != nil {
		return fmt.Errorf("setting reset required: %w", err)
	}

	slog.Info("bootstrap admin created — password change required on first login", "email", email)
	return nil
}

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
