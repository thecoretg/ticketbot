package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const sessionDuration = 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoPassword         = errors.New("user has no password set")
)

type Service struct {
	users    repos.APIUserRepository
	sessions repos.SessionRepository
}

func New(users repos.APIUserRepository, sessions repos.SessionRepository) *Service {
	return &Service{users: users, sessions: sessions}
}

// Login validates credentials and returns a new session token (hex-encoded, 32 bytes).
// The caller should store this token in a cookie; this service stores only the SHA-256 hash.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.users.GetForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			return "", ErrInvalidCredentials
		}
		return "", fmt.Errorf("looking up user: %w", err)
	}

	if len(u.PasswordHash) == 0 {
		return "", ErrNoPassword
	}

	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	token, hash, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generating session token: %w", err)
	}

	_, err = s.sessions.Create(ctx, &models.Session{
		UserID:    u.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(sessionDuration),
	})
	if err != nil {
		return "", fmt.Errorf("creating session: %w", err)
	}

	return token, nil
}

// ValidateToken looks up a session by the token from a cookie and returns the user ID.
func (s *Service) ValidateToken(ctx context.Context, token string) (int, error) {
	hash := hashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		return 0, err
	}

	return session.UserID, nil
}

// Logout deletes the session for the given token.
func (s *Service) Logout(ctx context.Context, token string) error {
	hash := hashToken(token)
	session, err := s.sessions.GetByTokenHash(ctx, hash)
	if err != nil {
		if errors.Is(err, models.ErrSessionNotFound) {
			return nil
		}
		return err
	}

	return s.sessions.Delete(ctx, session.ID)
}

// SetPassword sets (or resets) the bcrypt password hash for a user.
func (s *Service) SetPassword(ctx context.Context, userID int, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	return s.users.SetPassword(ctx, userID, hash)
}

func generateToken() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, err
	}

	token = hex.EncodeToString(raw)
	hash = hashToken(token)
	return token, hash, nil
}

func hashToken(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}
