package repos

import (
	"context"

	"github.com/thecoretg/ticketbot/models"
)

type SessionRepository interface {
	Create(ctx context.Context, s *models.Session) (*models.Session, error)
	GetByTokenHash(ctx context.Context, tokenHash []byte) (*models.Session, error)
	Delete(ctx context.Context, id int) error
	DeleteExpired(ctx context.Context) error
}
