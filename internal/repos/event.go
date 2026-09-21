package repos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type TicketEventRepository interface {
	WithTx(tx pgx.Tx) TicketEventRepository
	Insert(ctx context.Context, e *models.TicketEvent) (*models.TicketEvent, error)
	InsertBatch(ctx context.Context, events []*models.TicketEvent) error
	// ListByTicket returns up to limit events for a ticket, newest first. When beforeID is non-nil
	// only events with a smaller id are returned, for paging backwards through history.
	ListByTicket(ctx context.Context, ticketID int, limit int, beforeID *int64) ([]*models.TicketEvent, error)
	ListRecent(ctx context.Context, kind *models.EventKind, limit int) ([]*models.TicketEvent, error)
	// ListByRun returns every event of one run, oldest first.
	ListByRun(ctx context.Context, runID string) ([]*models.TicketEvent, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
}
