package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type TransformerRuleRepository interface {
	WithTx(tx pgx.Tx) TransformerRuleRepository
	ListAll(ctx context.Context) ([]*models.TransformerRule, error)
	ListEnabled(ctx context.Context) ([]*models.TransformerRule, error)
	Get(ctx context.Context, id int) (*models.TransformerRule, error)
	Exists(ctx context.Context, id int) (bool, error)
	Insert(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error)
	Update(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error)
	Delete(ctx context.Context, id int) error
}

type TransformerRunRepository interface {
	WithTx(tx pgx.Tx) TransformerRunRepository
	Exists(ctx context.Context, ticketID, ruleID int) (bool, error)
	Insert(ctx context.Context, ticketID, ruleID int) error
}
