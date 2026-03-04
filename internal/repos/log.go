package repos

import (
	"context"
	"time"

	"github.com/thecoretg/ticketbot/internal/logging"
)

type LogRepository interface {
	InsertBatch(ctx context.Context, entries []logging.LogEntry) error
	GetRecent(ctx context.Context, limit int) ([]logging.LogEntry, error)
	DeleteOlderThan(ctx context.Context, before time.Time) (int64, error)
}
