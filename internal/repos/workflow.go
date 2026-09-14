package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowRepository interface {
	WithTx(tx pgx.Tx) WorkflowRepository
	List(ctx context.Context) ([]*models.Workflow, error)
	Get(ctx context.Context, id int) (*models.Workflow, error)
	GetByBoard(ctx context.Context, boardID int) (*models.Workflow, error)
	ExistsForBoard(ctx context.Context, boardID int) (bool, error)
	Insert(ctx context.Context, w *models.Workflow) (*models.Workflow, error)
	// Update replaces name, enabled, dry_run and the whole rule list.
	Update(ctx context.Context, w *models.Workflow) (*models.Workflow, error)
	Delete(ctx context.Context, id int) error
}
