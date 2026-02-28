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
	GetForAuth(ctx context.Context, email string) (*models.UserAuth, error)
	Exists(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, email string) (*models.APIUser, error)
	GetForAuthByID(ctx context.Context, id int) (*models.UserAuth, error)
	SetPassword(ctx context.Context, id int, hash []byte) error
	SetPasswordResetRequired(ctx context.Context, id int, required bool) error
	SetTOTPSecret(ctx context.Context, id int, secret *string) error
	SetTOTPEnabled(ctx context.Context, id int, enabled bool) error
	Delete(ctx context.Context, id int) error
}
