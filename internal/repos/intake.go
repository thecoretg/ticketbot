package repos

import (
	"context"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// WebhookIntakeRepository is the persisted webhook queue.
type WebhookIntakeRepository interface {
	Insert(ctx context.Context, ticketID int, action models.IntakeAction, payload []byte) (*models.WebhookIntake, error)
	// Claim marks the next ready row processing and returns it, or ErrIntakeEmpty when nothing is
	// ready. A ticket with an older open row is never claimed, so one ticket's webhooks run in
	// arrival order and never concurrently.
	Claim(ctx context.Context) (*models.WebhookIntake, error)
	Finish(ctx context.Context, id int64) error
	Reschedule(ctx context.Context, id int64, at time.Time, lastError string) error
	Fail(ctx context.Context, id int64, lastError string) error
	// Retry and Discard act on failed rows only; both return ErrIntakeNotFound otherwise.
	Retry(ctx context.Context, id int64) (*models.WebhookIntake, error)
	Discard(ctx context.Context, id int64) (*models.WebhookIntake, error)
	// ResetProcessing returns rows a crash left in processing to pending.
	ResetProcessing(ctx context.Context) (int64, error)
	List(ctx context.Context, status *models.IntakeStatus, limit int) ([]*models.WebhookIntake, error)
	Stats(ctx context.Context) (*models.IntakeStats, error)
	CountReceivedSince(ctx context.Context, since time.Time) (int64, error)
	DeleteFinishedBefore(ctx context.Context, before time.Time) (int64, error)
}
