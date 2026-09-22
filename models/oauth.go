package models

import (
	"errors"
	"time"
)

var (
	ErrOAuthClientNotFound = errors.New("oauth client not found")
	ErrOAuthGrantNotFound  = errors.New("oauth grant not found")
	ErrOAuthCodeNotFound   = errors.New("oauth code not found")
	ErrOAuthTokenNotFound  = errors.New("oauth token not found")
)

// OAuthClient is a dynamically registered MCP client. ExpiresAt is set at registration and
// cleared by the first grant, so abandoned registrations age out.
type OAuthClient struct {
	ID           string     `json:"client_id"`
	Name         string     `json:"client_name"`
	RedirectURIs []string   `json:"redirect_uris"`
	CreatedOn    time.Time  `json:"created_on"`
	ExpiresAt    *time.Time `json:"-"`
}

// OAuthGrant is a user's approval of a client for a set of scopes. ClientName is filled by
// list queries for the Connected apps page.
type OAuthGrant struct {
	ID         int        `json:"id"`
	UserID     int        `json:"user_id"`
	ClientID   string     `json:"client_id"`
	ClientName string     `json:"client_name,omitempty"`
	Scopes     []string   `json:"scopes"`
	CreatedOn  time.Time  `json:"created_on"`
	LastUsedAt *time.Time `json:"last_used_at"`
}

// OAuthCode is a one-shot authorization code awaiting exchange.
type OAuthCode struct {
	CodeHash      []byte
	GrantID       int
	RedirectURI   string
	CodeChallenge string
	ExpiresAt     time.Time
}

// OAuthTokenKind distinguishes the two token rows a grant issues.
type OAuthTokenKind string

const (
	OAuthAccessToken  OAuthTokenKind = "access"
	OAuthRefreshToken OAuthTokenKind = "refresh"
)

// OAuthToken is an issued access or refresh token. UsedAt is set on a refresh token once it
// has been rotated; seeing it again means replay.
type OAuthToken struct {
	ID        int
	GrantID   int
	Kind      OAuthTokenKind
	TokenHash []byte
	ExpiresAt time.Time
	CreatedOn time.Time
	UsedAt    *time.Time
}

// OAuthAccess is what a bearer access token resolves to.
type OAuthAccess struct {
	GrantID   int
	UserID    int
	Scopes    []string
	ExpiresAt time.Time
}

// OAuthPurgeCounts reports what the hourly sweep removed.
type OAuthPurgeCounts struct {
	Codes   int64
	Tokens  int64
	Clients int64
}
