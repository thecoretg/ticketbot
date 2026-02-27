package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type ConfigRepository interface {
	WithTx(tx pgx.Tx) ConfigRepository
	Get(ctx context.Context) (*models.Config, error)
	InsertDefault(ctx context.Context) (*models.Config, error)
	Upsert(ctx context.Context, c *models.Config) (*models.Config, error)
}
