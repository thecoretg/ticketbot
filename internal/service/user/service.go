package user

import (
	"context"

	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Users models.APIUserRepository
	Keys  models.APIKeyRepository
}

func New(u models.APIUserRepository, k models.APIKeyRepository) *Service {
	return &Service{
		Users: u,
		Keys:  k,
	}
}

func (s *Service) ListUsers(ctx context.Context) ([]models.APIUser, error) {
	return s.Users.List(ctx)
}

func (s *Service) GetUser(ctx context.Context, id int) (*models.APIUser, error) {
	return s.Users.Get(ctx, id)
}

func (s *Service) DeleteUser(ctx context.Context, id int) error {
	return s.Users.Delete(ctx, id)
}

func (s *Service) ListAPIKeys(ctx context.Context) ([]models.APIKey, error) {
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
