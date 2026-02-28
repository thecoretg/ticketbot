package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"
	"unicode"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	sessionDuration     = 24 * time.Hour
	totpPendingDuration = 5 * time.Minute
	recoveryCodeCount   = 10
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrNoPassword         = errors.New("user has no password set")
	ErrWeakPassword       = errors.New("password must be at least 8 characters and include an uppercase letter, lowercase letter, and number")
	ErrInvalidTOTPCode    = errors.New("invalid or expired TOTP code")
)

// LoginResult is returned by Login. When TOTPRequired is true the caller must
// redirect to the TOTP verification step using PendingToken.
type LoginResult struct {
	TOTPRequired  bool
	Token         string // session token (only set when TOTPRequired is false)
	PendingToken  string // pending token (only set when TOTPRequired is true)
	ResetRequired bool   // only meaningful when TOTPRequired is false
}

// ValidatePassword enforces password strength requirements.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrWeakPassword
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}

	if !hasUpper || !hasLower || !hasDigit {
		return ErrWeakPassword
	}

	return nil
}

type Service struct {
	users        repos.APIUserRepository
	sessions     repos.SessionRepository
	totpPending  repos.TOTPPendingRepository
	totpRecovery repos.TOTPRecoveryRepository
}

func New(users repos.APIUserRepository, sessions repos.SessionRepository, totpPending repos.TOTPPendingRepository, totpRecovery repos.TOTPRecoveryRepository) *Service {
	return &Service{users: users, sessions: sessions, totpPending: totpPending, totpRecovery: totpRecovery}
}

// Login validates credentials. If the user has TOTP enabled it returns a
// short-lived pending token that must be exchanged via VerifyTOTP; otherwise
// it creates a full session and returns the session token.
func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	u, err := s.users.GetForAuth(ctx, email)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("looking up user: %w", err)
	}

	if len(u.PasswordHash) == 0 {
		return LoginResult{}, ErrNoPassword
	}

	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return LoginResult{}, ErrInvalidCredentials
	}

	if u.TOTPEnabled {
		pendingToken, hash, err := generateToken()
		if err != nil {
			return LoginResult{}, fmt.Errorf("generating pending token: %w", err)
		}

		_, err = s.totpPending.Create(ctx, &models.TOTPPending{
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: time.Now().Add(totpPendingDuration),
		})
		if err != nil {
			return LoginResult{}, fmt.Errorf("creating pending token: %w", err)
		}

		return LoginResult{TOTPRequired: true, PendingToken: pendingToken}, nil
	}

	token, hash, err := generateToken()
	if err != nil {
		return LoginResult{}, fmt.Errorf("generating session token: %w", err)
	}

	_, err = s.sessions.Create(ctx, &models.Session{
		UserID:    u.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(sessionDuration),
	})
	if err != nil {
		return LoginResult{}, fmt.Errorf("creating session: %w", err)
	}

	return LoginResult{Token: token, ResetRequired: u.ResetRequired}, nil
}

// ChangePassword validates the current password, sets the new one, and clears the reset flag.
func (s *Service) ChangePassword(ctx context.Context, userID int, currentPwd, newPwd string) error {
	u, err := s.users.GetForAuthByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("looking up user: %w", err)
	}

	if len(u.PasswordHash) == 0 {
		return ErrNoPassword
	}

	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(currentPwd)); err != nil {
		return ErrInvalidCredentials
	}

	if err := ValidatePassword(newPwd); err != nil {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPwd), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hashing new password: %w", err)
	}

	if err := s.users.SetPassword(ctx, userID, hash); err != nil {
		return fmt.Errorf("storing new password: %w", err)
	}

	if err := s.users.SetPasswordResetRequired(ctx, userID, false); err != nil {
		return fmt.Errorf("clearing reset flag: %w", err)
	}

	return nil
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
