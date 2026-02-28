package user

import (
	"context"
	"fmt"

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

type Service struct {
	Users repos.APIUserRepository
	Keys  repos.APIKeyRepository
}

func New(u repos.APIUserRepository, k repos.APIKeyRepository) *Service {
	return &Service{
		Users: u,
		Keys:  k,
	}
}

func (s *Service) ListUsers(ctx context.Context) ([]*models.APIUser, error) {
	return s.Users.List(ctx)
}

func (s *Service) GetUser(ctx context.Context, id int) (*models.APIUser, error) {
	return s.Users.Get(ctx, id)
}

func (s *Service) GetUserByEmail(ctx context.Context, email string) (*models.APIUser, error) {
	return s.Users.GetByEmail(ctx, email)
}

func (s *Service) InsertUser(ctx context.Context, email string) (*models.APIUser, error) {
	exists, err := s.Users.Exists(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("checking if user exists: %w", err)
	}

	if exists {
		return nil, ErrUserAlreadyExists{Email: email}
	}

	return s.Users.Insert(ctx, email)
}

// InsertUserWithPassword creates a user and sets a temporary password that must be changed on first login.
func (s *Service) InsertUserWithPassword(ctx context.Context, email, password string) (*models.APIUser, error) {
	u, err := s.InsertUser(ctx, email)
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
	return s.Users.Delete(ctx, id)
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
