package models

import (
	"errors"
	"time"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

type CreateAPIKeyPayload struct {
	Email string `json:"email"`
}

type CreateAPIKeyResponse struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

type APIKey struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	KeyHash   []byte    `json:"key_hash"`
	KeyHint   *string   `json:"key_hint,omitempty"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

var ErrAPIUserNotFound = errors.New("api user not found")

type APIUser struct {
	ID           int    `json:"id"`
	EmailAddress string `json:"email_address"`
	Role         Role   `json:"role"`
	// SSO is true when the account is linked to a Microsoft Entra identity. Such accounts take
	// their role from Entra on every sign-in and have no local password.
	SSO bool `json:"sso"`
	// BreakGlass marks the INITIAL_ADMIN_EMAIL account: it can always sign in with a password,
	// so it cannot be deleted, demoted or handed to Entra.
	BreakGlass bool      `json:"break_glass"`
	CreatedOn  time.Time `json:"created_on"`
	UpdatedOn  time.Time `json:"updated_on"`
}

// UserAuth is a restricted view of APIUser used only during login.
// It is separate from APIUser to preserve comparability ([]byte is not comparable).
type UserAuth struct {
	ID            int
	EmailAddress  string
	PasswordHash  []byte
	ResetRequired bool
	TOTPSecret    *string
	TOTPEnabled   bool
	Role          Role
	EntraOID      *string
}
