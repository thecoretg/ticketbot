package authsvc

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pquerna/otp/totp"
	qrcode "github.com/skip2/go-qrcode"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

// BeginSetup generates a new TOTP secret and returns the secret (base32),
// the otpauth:// URL, and a QR code PNG for display. The secret is NOT stored
// yet — the user must confirm a valid code via ConfirmSetup first.
func (s *Service) BeginSetup(ctx context.Context, userID int) (secret, otpauthURL string, qrPNG []byte, err error) {
	u, err := s.users.Get(ctx, userID)
	if err != nil {
		return "", "", nil, fmt.Errorf("looking up user: %w", err)
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "TicketBot",
		AccountName: u.EmailAddress,
	})
	if err != nil {
		return "", "", nil, fmt.Errorf("generating TOTP key: %w", err)
	}

	png, err := qrcode.Encode(key.URL(), qrcode.Medium, 256)
	if err != nil {
		return "", "", nil, fmt.Errorf("generating QR code: %w", err)
	}

	return key.Secret(), key.URL(), png, nil
}

// ConfirmSetup validates the user's password and a TOTP code against the
// provided secret (from BeginSetup). On success it enables TOTP, stores the
// secret, generates fresh recovery codes, and returns them (shown once).
func (s *Service) ConfirmSetup(ctx context.Context, userID int, password, code, secret string) ([]string, error) {
	code = strings.ReplaceAll(code, " ", "")

	u, err := s.users.GetForAuthByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		slog.Error("totp confirm: password check failed", "user_id", userID)
		return nil, ErrInvalidCredentials
	}

	if !totp.Validate(code, secret) {
		expected, genErr := totp.GenerateCode(secret, time.Now().UTC())
		slog.Error("totp confirm: TOTP code invalid",
			"user_id", userID,
			"code_len", len(code),
			"secret_len", len(secret),
			"gen_err", genErr,
			"codes_match", expected == code,
		)
		return nil, ErrInvalidTOTPCode
	}

	if err := s.users.SetTOTPSecret(ctx, userID, &secret); err != nil {
		return nil, fmt.Errorf("storing TOTP secret: %w", err)
	}
	if err := s.users.SetTOTPEnabled(ctx, userID, true); err != nil {
		return nil, fmt.Errorf("enabling TOTP: %w", err)
	}

	if err := s.totpRecovery.DeleteAll(ctx, userID); err != nil {
		return nil, fmt.Errorf("clearing old recovery codes: %w", err)
	}

	return s.generateRecoveryCodes(ctx, userID)
}

// VerifyTOTP validates a TOTP code (or recovery code) against a pending token
// produced by Login. On success it deletes the pending token and creates a
// real session.
func (s *Service) VerifyTOTP(ctx context.Context, pendingToken, code string) (sessionToken string, resetRequired bool, recoveryCodeUsed bool, err error) {
	// Strip any spaces (some authenticator apps display codes as "123 456").
	code = strings.ReplaceAll(code, " ", "")

	tokenHash := hashToken(pendingToken)
	pending, err := s.totpPending.GetByTokenHash(ctx, tokenHash)
	if err != nil {
		slog.Error("totp verify: pending token not found", "err", err)
		return "", false, false, ErrInvalidCredentials
	}

	u, err := s.users.GetForAuthByID(ctx, pending.UserID)
	if err != nil {
		return "", false, false, fmt.Errorf("looking up user: %w", err)
	}

	if !u.TOTPEnabled || u.TOTPSecret == nil {
		slog.Error("totp verify: totp not configured on account", "user_id", pending.UserID, "enabled", u.TOTPEnabled, "has_secret", u.TOTPSecret != nil)
		return "", false, false, ErrInvalidCredentials
	}

	if !totp.Validate(code, *u.TOTPSecret) {
		slog.Error("totp verify: TOTP code invalid, trying recovery code", "user_id", pending.UserID)
		// Try as a recovery code.
		codeHash := sha256Code(code)
		rc, err := s.totpRecovery.GetUnusedByHash(ctx, pending.UserID, codeHash)
		if err != nil {
			slog.Error("totp verify: recovery code not found either", "user_id", pending.UserID)
			return "", false, false, ErrInvalidCredentials
		}
		if err := s.totpRecovery.MarkUsed(ctx, rc.ID); err != nil {
			return "", false, false, fmt.Errorf("marking recovery code used: %w", err)
		}
		recoveryCodeUsed = true
	}

	if err := s.totpPending.Delete(ctx, pending.ID); err != nil {
		return "", false, false, fmt.Errorf("deleting pending token: %w", err)
	}

	token, hash, err := generateToken()
	if err != nil {
		return "", false, false, fmt.Errorf("generating session token: %w", err)
	}

	_, err = s.sessions.Create(ctx, &models.Session{
		UserID:    pending.UserID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(sessionDuration),
	})
	if err != nil {
		return "", false, false, fmt.Errorf("creating session: %w", err)
	}

	return token, u.ResetRequired, recoveryCodeUsed, nil
}

// TOTPStatus returns whether TOTP is currently enabled for the user.
func (s *Service) TOTPStatus(ctx context.Context, userID int) (bool, error) {
	u, err := s.users.GetForAuthByID(ctx, userID)
	if err != nil {
		return false, fmt.Errorf("looking up user: %w", err)
	}
	return u.TOTPEnabled, nil
}

// DisableTOTP validates the user's password then clears TOTP from the account.
func (s *Service) DisableTOTP(ctx context.Context, userID int, password string) error {
	u, err := s.users.GetForAuthByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("looking up user: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword(u.PasswordHash, []byte(password)); err != nil {
		return ErrInvalidCredentials
	}

	if err := s.users.SetTOTPEnabled(ctx, userID, false); err != nil {
		return fmt.Errorf("disabling TOTP: %w", err)
	}
	if err := s.users.SetTOTPSecret(ctx, userID, nil); err != nil {
		return fmt.Errorf("clearing TOTP secret: %w", err)
	}
	if err := s.totpRecovery.DeleteAll(ctx, userID); err != nil {
		return fmt.Errorf("clearing recovery codes: %w", err)
	}

	return nil
}

func (s *Service) generateRecoveryCodes(ctx context.Context, userID int) ([]string, error) {
	codes := make([]string, recoveryCodeCount)
	for i := range codes {
		raw := make([]byte, 10)
		if _, err := rand.Read(raw); err != nil {
			return nil, fmt.Errorf("generating recovery code: %w", err)
		}
		code := hex.EncodeToString(raw)
		codes[i] = code

		if err := s.totpRecovery.Insert(ctx, userID, sha256Code(code)); err != nil {
			return nil, fmt.Errorf("storing recovery code: %w", err)
		}
	}
	return codes, nil
}

func sha256Code(code string) []byte {
	h := sha256.Sum256([]byte(code))
	return h[:]
}
