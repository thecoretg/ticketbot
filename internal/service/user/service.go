package user

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

type ErrUserAlreadyExists struct {
	Email string
}

func (e ErrUserAlreadyExists) Error() string {
	return fmt.Sprintf("user with email '%s' already exists", e.Email)
}

type ErrCannotDeleteSelf struct{}

func (e ErrCannotDeleteSelf) Error() string {
	return "cannot delete your own user account"
}

type ErrCannotChangeOwnRole struct{}

func (e ErrCannotChangeOwnRole) Error() string {
	return "cannot change your own role"
}

// ErrRoleManagedByEntra is returned when changing the role of an SSO-linked account, whose role
// is re-derived from Entra on every sign-in.
type ErrRoleManagedByEntra struct{}

func (e ErrRoleManagedByEntra) Error() string {
	return "this account signs in with Microsoft; change its role in Entra"
}

// ErrBreakGlassProtected is returned when a change would weaken the INITIAL_ADMIN_EMAIL account,
// the one guaranteed password sign-in when SSO is on.
type ErrBreakGlassProtected struct{}

func (e ErrBreakGlassProtected) Error() string {
	return "this is the break-glass admin account set by INITIAL_ADMIN_EMAIL; it cannot be deleted or demoted"
}

type Service struct {
	Users repos.APIUserRepository
	Keys  repos.APIKeyRepository
	// breakGlassEmail is INITIAL_ADMIN_EMAIL.
	breakGlassEmail string
}

func New(u repos.APIUserRepository, k repos.APIKeyRepository, breakGlassEmail string) *Service {
	return &Service{
		Users:           u,
		Keys:            k,
		breakGlassEmail: breakGlassEmail,
	}
}

// IsBreakGlass reports whether email is the INITIAL_ADMIN_EMAIL account.
func (s *Service) IsBreakGlass(email string) bool {
	return s.breakGlassEmail != "" && strings.EqualFold(email, s.breakGlassEmail)
}

func (s *Service) mark(u *models.APIUser) *models.APIUser {
	if u != nil {
		u.BreakGlass = s.IsBreakGlass(u.EmailAddress)
	}
	return u
}

func (s *Service) ListUsers(ctx context.Context) ([]*models.APIUser, error) {
	users, err := s.Users.List(ctx)
	for _, u := range users {
		s.mark(u)
	}
	return users, err
}

func (s *Service) GetUser(ctx context.Context, id int) (*models.APIUser, error) {
	u, err := s.Users.Get(ctx, id)
	return s.mark(u), err
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.APIUser, error) {
	return s.Users.GetByEmail(ctx, email)
}

func (s *Service) InsertUser(ctx context.Context, email string, role models.Role) (*models.APIUser, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("%w: %q", models.ErrInvalidRole, role)
	}
	exists, err := s.Users.Exists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("checking if user exists: %w", err)
	}

	if exists {
		return nil, ErrUserAlreadyExists{Email: email}
	}

	return s.Users.Insert(ctx, email, role)
}

// InsertUserWithPassword creates a user and sets a temporary password that must be changed on first login.
func (s *Service) InsertUserWithPassword(ctx context.Context, email, password string, role models.Role) (*models.APIUser, error) {
	u, err := s.InsertUser(ctx, email, role)
	if err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	if err := s.Users.SetPassword(ctx, u.ID, hash); err != nil {
		return nil, fmt.Errorf("setting password: %w", err)
	}

	if err := s.Users.SetPasswordResetRequired(ctx, u.ID, true); err != nil {
		return nil, fmt.Errorf("setting reset flag: %w", err)
	}

	return u, nil
}

func (s *Service) DeleteUser(ctx context.Context, id int, authenticatedUserID int) error {
	if id == authenticatedUserID {
		return ErrCannotDeleteSelf{}
	}
	u, err := s.Users.Get(ctx, id)
	if err != nil {
		return err
	}
	if s.IsBreakGlass(u.EmailAddress) {
		return ErrBreakGlassProtected{}
	}
	return s.Users.Delete(ctx, id)
}

// SetRole changes a local account's role. Your own role and Entra-managed roles are refused.
func (s *Service) SetRole(ctx context.Context, id int, role models.Role, authenticatedUserID int) (*models.APIUser, error) {
	if !role.Valid() {
		return nil, fmt.Errorf("%w: %q", models.ErrInvalidRole, role)
	}
	if id == authenticatedUserID {
		return nil, ErrCannotChangeOwnRole{}
	}
	u, err := s.Users.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	if u.SSO {
		return nil, ErrRoleManagedByEntra{}
	}
	if s.IsBreakGlass(u.EmailAddress) {
		return nil, ErrBreakGlassProtected{}
	}
	if err := s.Users.SetRole(ctx, id, role); err != nil {
		return nil, fmt.Errorf("setting role: %w", err)
	}
	return s.Users.Get(ctx, id)
}

func (s *Service) ListAPIKeys(ctx context.Context) ([]*models.APIKey, error) {
	return s.Keys.List(ctx)
}

func (s *Service) GetAPIKey(ctx context.Context, id int) (*models.APIKey, error) {
	return s.Keys.Get(ctx, id)
}

// AddAPIKey creates an API key and returns the plaintext (only once)
func (s *Service) AddAPIKey(ctx context.Context, email string) (string, error) {
	return s.createAPIKey(ctx, email, nil)
}

func (s *Service) DeleteAPIKey(ctx context.Context, id int) error {
	return s.Keys.Delete(ctx, id)
}
