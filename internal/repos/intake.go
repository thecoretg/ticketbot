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
	// CountsByHour returns one row per hour that received at least one webhook since the given time.
	CountsByHour(ctx context.Context, since time.Time) ([]models.IntakeHourCount, error)
	DeleteFinishedBefore(ctx context.Context, before time.Time) (int64, error)
	// OpenTickets returns which of ticketIDs have a pending or processing row.
	OpenTickets(ctx context.Context, ticketIDs []int) ([]int, error)
	CountActionSince(ctx context.Context, action models.IntakeAction, since time.Time) (int64, error)
}

// CatchupStateRepository holds the missed-webhook check's single state row. Get returns nil and
// no error before the first run.
type CatchupStateRepository interface {
	Get(ctx context.Context) (*models.CatchupState, error)
	Save(ctx context.Context, st *models.CatchupState) error
}

// ScheduledJobRepository keeps one record per clock-driven job. Get returns nil and no error
// before a job's first run.
type ScheduledJobRepository interface {
	Get(ctx context.Context, name string) (*models.ScheduledJob, error)
	Save(ctx context.Context, j *models.ScheduledJob) error
}
