package user

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
	"golang.org/x/crypto/bcrypt"
)

func (s *Service) createAPIKey(ctx context.Context, email string) (string, error) {
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

	hash, err := hashKey(plain)
	if err != nil {
		return "", fmt.Errorf("hashing key: %w", err)
	}

	p := &models.APIKey{
		UserID:  u.ID,
		KeyHash: hash,
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
