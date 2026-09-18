// Package sso adapts Microsoft Entra ID sign-in (tctg-go/entra) to ticketbot's users: it decides
// who may sign in from their Entra app roles, provisions accounts just in time, and exposes the
// settings the dashboard's single sign-on page shows.
package sso

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

const (
	// CallbackPath is where Microsoft redirects after sign-in; RedirectURI is ROOT_URL + this.
	CallbackPath = "/auth/sso/callback"
	// PanelPath is the dashboard, which doubles as the login page and the post-login landing.
	PanelPath = "/"
	// SessionCookie is the name entra gives its session cookie (its default).
	SessionCookie = "entra_session"
)

var (
	ErrNotConfigured = errors.New("sso is not configured: set ENTRA_TENANT_ID, ENTRA_CLIENT_ID and ENTRA_CLIENT_SECRET")
	ErrEmptyRole     = errors.New("entra role must not be empty")
	// ErrEmailLinked means the email in the token belongs to a local account already linked to a
	// different Entra identity.
	ErrEmailLinked = errors.New("email already linked to another microsoft account")
)

// noMappedRole is what a person sees when none of their app roles maps to a ticketbot role.
const noMappedRole = "Your account has no app role that grants access to ticketbot."

type Params struct {
	Users    repos.APIUserRepository
	Mappings repos.SSORoleMappingRepository
	Cfg      *models.Config
	Entra    env.Entra
	RootURL  string
}

// Service is both the entra.Authorizer and the entra.Provisioner for ticketbot. Role mappings
// are cached because entra.Authorizer receives no context; Reload runs after every change.
type Service struct {
	users    repos.APIUserRepository
	mappings repos.SSORoleMappingRepository
	cfg      *models.Config
	entra    env.Entra
	rootURL  string

	// auth is nil until SetAuth, and stays nil when SSO is not configured.
	auth *entra.Auth[*models.APIUser]

	mu          sync.RWMutex
	byEntraRole map[string]models.Role
}

// New builds the service and loads the role mappings.
func New(ctx context.Context, p Params) (*Service, error) {
	s := &Service{
		users:       p.Users,
		mappings:    p.Mappings,
		cfg:         p.Cfg,
		entra:       p.Entra,
		rootURL:     p.RootURL,
		byEntraRole: map[string]models.Role{},
	}
	if err := s.Reload(ctx); err != nil {
		return nil, err
	}
	return s, nil
}

// SetAuth hands the service the entra.Auth built around it.
func (s *Service) SetAuth(a *entra.Auth[*models.APIUser]) { s.auth = a }

// Configured reports whether the ENTRA_* variables are present.
func (s *Service) Configured() bool { return s.auth != nil }

// Enabled reports whether sign-in with Microsoft is currently offered.
func (s *Service) Enabled() bool { return s.Configured() && s.cfg.SSOEnabled }

// Methods tells the login page which sign-in options to show. Password sign-in is always offered
// when SSO is not, whatever the toggle says, so removing the ENTRA_* variables is a way back in.
func (s *Service) Methods() models.AuthMethods {
	sso := s.Enabled()
	return models.AuthMethods{SSO: sso, Password: s.cfg.PasswordLoginEnabled || !sso}
}

// Reload refreshes the cached role mappings from the store.
func (s *Service) Reload(ctx context.Context) error {
	ms, err := s.mappings.List(ctx)
	if err != nil {
		return fmt.Errorf("loading sso role mappings: %w", err)
	}
	m := make(map[string]models.Role, len(ms))
	for _, r := range ms {
		m[r.EntraRole] = r.Role
	}
	s.mu.Lock()
	s.byEntraRole = m
	s.mu.Unlock()
	return nil
}

// resolveRole returns the most privileged ticketbot role granted by any of the Entra app roles.
func (s *Service) resolveRole(entraRoles []string) (models.Role, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var granted []models.Role
	for _, r := range entraRoles {
		if role, ok := s.byEntraRole[r]; ok {
			granted = append(granted, role)
		}
	}
	return models.HighestRole(granted)
}

// Authorize is the entra.Authorizer: sign-in is refused unless at least one app role maps.
func (s *Service) Authorize(c entra.Claims) error {
	if _, ok := s.resolveRole(c.Roles); !ok {
		return &entra.DeniedError{Message: noMappedRole}
	}
	return nil
}

// Provision implements entra.Provisioner. It links on the immutable object ID; a local account
// with the same email and no link yet is adopted once, so the bootstrap admin can move to SSO.
// The role is re-derived from Entra on every sign-in.
func (s *Service) Provision(ctx context.Context, c entra.Claims) (string, error) {
	role, ok := s.resolveRole(c.Roles)
	if !ok {
		return "", &entra.DeniedError{Message: noMappedRole}
	}
	email := c.PreferredUsername
	if email == "" {
		email = c.Email
	}
	if email == "" {
		return "", errors.New("id token carries neither preferred_username nor email")
	}

	u, err := s.users.GetByEntraOID(ctx, c.OID)
	switch {
	case err == nil:
		if u.Role != role || u.EmailAddress != email {
			if err := s.users.LinkEntra(ctx, u.ID, c.OID, email); err != nil {
				return "", fmt.Errorf("refreshing linked user: %w", err)
			}
			if err := s.users.SetRole(ctx, u.ID, role); err != nil {
				return "", fmt.Errorf("refreshing linked user role: %w", err)
			}
		}
		return strconv.Itoa(u.ID), nil
	case !errors.Is(err, models.ErrAPIUserNotFound):
		return "", fmt.Errorf("looking up user by entra oid: %w", err)
	}

	u, err = s.users.GetByEmailFold(ctx, email)
	switch {
	case err == nil:
		if u.SSO {
			return "", ErrEmailLinked
		}
	case errors.Is(err, models.ErrAPIUserNotFound):
		u, err = s.users.Insert(ctx, email, role)
		if err != nil {
			return "", fmt.Errorf("creating user: %w", err)
		}
	default:
		return "", fmt.Errorf("looking up user by email: %w", err)
	}

	if err := s.users.LinkEntra(ctx, u.ID, c.OID, email); err != nil {
		return "", fmt.Errorf("linking user to entra: %w", err)
	}
	if err := s.users.SetRole(ctx, u.ID, role); err != nil {
		return "", fmt.Errorf("setting role: %w", err)
	}
	return strconv.Itoa(u.ID), nil
}

// Lookup implements entra.Provisioner. It runs on every authenticated request.
func (s *Service) Lookup(ctx context.Context, userID string) (*models.APIUser, error) {
	id, err := strconv.Atoi(userID)
	if err != nil {
		return nil, entra.ErrUserNotFound
	}
	u, err := s.users.Get(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			return nil, entra.ErrUserNotFound
		}
		return nil, err
	}
	return u, nil
}

// RedirectURI is the value to register on the app registration.
func (s *Service) RedirectURI() string {
	if s.auth != nil {
		return s.auth.RedirectURI()
	}
	return s.rootURL + CallbackPath
}

// Status is what the dashboard's SSO page shows.
func (s *Service) Status(ctx context.Context) (*models.SSOStatus, error) {
	ms, err := s.mappings.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing sso role mappings: %w", err)
	}
	return &models.SSOStatus{
		Configured:           s.Configured(),
		Enabled:              s.Enabled(),
		PasswordLoginEnabled: s.cfg.PasswordLoginEnabled,
		RedirectURI:          s.RedirectURI(),
		TenantID:             s.entra.TenantID,
		ClientID:             s.entra.ClientID,
		Mappings:             ms,
	}, nil
}

// TestConnection proves the tenant is reachable and the client secret is accepted.
func (s *Service) TestConnection(ctx context.Context) error {
	if s.auth == nil {
		return ErrNotConfigured
	}
	return s.auth.TestConnection(ctx)
}

func (s *Service) UpsertMapping(ctx context.Context, entraRole string, role models.Role) (*models.SSORoleMapping, error) {
	entraRole = strings.TrimSpace(entraRole)
	if entraRole == "" {
		return nil, ErrEmptyRole
	}
	if !role.Valid() {
		return nil, fmt.Errorf("%w: %q", models.ErrInvalidRole, role)
	}
	m, err := s.mappings.Upsert(ctx, entraRole, role)
	if err != nil {
		return nil, fmt.Errorf("saving sso role mapping: %w", err)
	}
	return m, s.Reload(ctx)
}

func (s *Service) DeleteMapping(ctx context.Context, id int) error {
	if err := s.mappings.Delete(ctx, id); err != nil {
		return fmt.Errorf("deleting sso role mapping: %w", err)
	}
	return s.Reload(ctx)
}

// EndSession revokes the Entra session named by a cookie value. entra keys its stores by the
// hex SHA-256 of the cookie, as documented on entra.SessionStore.
func (s *Service) EndSession(ctx context.Context, cookie string) error {
	if s.auth == nil || cookie == "" {
		return nil
	}
	sum := sha256.Sum256([]byte(cookie))
	return s.auth.Sessions().DeleteSession(ctx, hex.EncodeToString(sum[:]))
}
