package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowRepository interface {
	WithTx(tx pgx.Tx) WorkflowRepository
	ListAll(ctx context.Context) ([]*models.Workflow, error)
	ListEnabled(ctx context.Context) ([]*models.Workflow, error)
	Get(ctx context.Context, id int) (*models.Workflow, error)
	Exists(ctx context.Context, id int) (bool, error)
	Insert(ctx context.Context, w *models.Workflow) (*models.Workflow, error)
	Update(ctx context.Context, w *models.Workflow) (*models.Workflow, error)
	Delete(ctx context.Context, id int) error
}

type WorkflowRunRepository interface {
	WithTx(tx pgx.Tx) WorkflowRunRepository
	Exists(ctx context.Context, ticketID, workflowID, actionIndex int) (bool, error)
	Insert(ctx context.Context, ticketID, workflowID, actionIndex int) error
}
