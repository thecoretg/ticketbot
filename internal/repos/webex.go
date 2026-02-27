package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type WebexRecipientRepository interface {
	WithTx(tx pgx.Tx) WebexRecipientRepository
	List(ctx context.Context) ([]*models.WebexRecipient, error)
	ListRooms(ctx context.Context) ([]*models.WebexRecipient, error)
	ListPeople(ctx context.Context) ([]*models.WebexRecipient, error)
	ListByEmail(ctx context.Context, email string) ([]*models.WebexRecipient, error)
	Get(ctx context.Context, id int) (*models.WebexRecipient, error)
	GetByWebexID(ctx context.Context, webexID string) (*models.WebexRecipient, error)
	Upsert(ctx context.Context, r *models.WebexRecipient) (*models.WebexRecipient, error)
	Delete(ctx context.Context, id int) error
}
