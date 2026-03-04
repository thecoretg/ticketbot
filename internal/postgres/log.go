package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/logging"
)

type LogRepo struct {
	pool *pgxpool.Pool
}

func NewLogRepo(pool *pgxpool.Pool) *LogRepo {
	return &LogRepo{pool: pool}
}

func (r *LogRepo) InsertBatch(ctx context.Context, entries []logging.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	rows := make([][]any, len(entries))
	for i, e := range entries {
		var attrsJSON []byte
		if e.Attrs != nil {
			var err error
			attrsJSON, err = json.Marshal(e.Attrs)
			if err != nil {
				attrsJSON = nil
			}
		}
		rows[i] = []any{e.Time, e.Level, e.Message, attrsJSON}
	}

	src := pgxCopyRows(rows)
	_, err := r.pool.CopyFrom(
		ctx,
		[]string{"app_log"},
		[]string{"time", "level", "message", "attrs"},
		&src,
	)
	return err
}

func (r *LogRepo) GetRecent(ctx context.Context, limit int) ([]logging.LogEntry, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT time, level, message, attrs
		FROM app_log
		ORDER BY time DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []logging.LogEntry
	for rows.Next() {
		var e logging.LogEntry
		var attrsJSON []byte
		if err := rows.Scan(&e.Time, &e.Level, &e.Message, &attrsJSON); err != nil {
			return nil, err
		}
		if attrsJSON != nil {
			_ = json.Unmarshal(attrsJSON, &e.Attrs)
		}
		entries = append(entries, e)
	}

	// reverse so entries are oldest-first
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}

	return entries, rows.Err()
}

func (r *LogRepo) DeleteOlderThan(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.pool.Exec(ctx, `DELETE FROM app_log WHERE time < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// pgxCopyRows adapts a [][]any slice to pgx's CopyFromSource interface.
type pgxCopyRows [][]any

func (r pgxCopyRows) Next() bool             { return len(r) > 0 }
func (r *pgxCopyRows) Values() ([]any, error) {
	row := (*r)[0]
	*r = (*r)[1:]
	return row, nil
}
func (r pgxCopyRows) Err() error { return nil }
