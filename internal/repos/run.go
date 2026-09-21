package repos

import (
	"context"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// WorkflowRunRepository stores the per-run summaries the results page lists.
type WorkflowRunRepository interface {
	Insert(ctx context.Context, r *models.WorkflowRun) error
	// Get returns models.ErrRunNotFound for an unknown id.
	Get(ctx context.Context, runID string) (*models.WorkflowRun, error)
	List(ctx context.Context, f models.RunFilter) ([]*models.WorkflowRun, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
