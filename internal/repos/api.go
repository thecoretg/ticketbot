package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type APIKeyRepository interface {
	WithTx(tx pgx.Tx) APIKeyRepository
	List(ctx context.Context) ([]*models.APIKey, error)
	Get(ctx context.Context, id int) (*models.APIKey, error)
	Insert(ctx context.Context, a *models.APIKey) (*models.APIKey, error)
	Delete(ctx context.Context, id int) error
}

type APIUserRepository interface {
	WithTx(tx pgx.Tx) APIUserRepository
	List(ctx context.Context) ([]*models.APIUser, error)
	Get(ctx context.Context, id int) (*models.APIUser, error)
	GetByEmail(ctx context.Context, email string) (*models.APIUser, error)
	Exists(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, email string) (*models.APIUser, error)
	Delete(ctx context.Context, id int) error
}
