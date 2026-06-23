package repos

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
)

type TicketJournalRepository interface {
	WithTx(tx pgx.Tx) TicketJournalRepository
	Get(ctx context.Context, ticketID int) (*models.TicketJournal, error)
	ListSummaries(ctx context.Context, limit int) ([]*models.TicketJournal, error)
	Upsert(ctx context.Context, j *models.TicketJournal) (*models.TicketJournal, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
