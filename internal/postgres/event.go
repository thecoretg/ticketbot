package postgres

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type copyFromer interface {
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

type TicketEventRepo struct {
	queries *db.Queries
	copier  copyFromer
}

func NewTicketEventRepo(pool *pgxpool.Pool) *TicketEventRepo {
	return &TicketEventRepo{
		queries: db.New(pool),
		copier:  pool,
	}
}

func (p *TicketEventRepo) WithTx(tx pgx.Tx) repos.TicketEventRepository {
	return &TicketEventRepo{
		queries: db.New(tx),
		copier:  tx,
	}
}

func (p *TicketEventRepo) Insert(ctx context.Context, e *models.TicketEvent) (*models.TicketEvent, error) {
	d, err := p.queries.InsertTicketEvent(ctx, db.InsertTicketEventParams{
		TicketID:   e.TicketID,
		RunID:      e.RunID,
		Kind:       string(e.Kind),
		Source:     string(e.Source),
		DryRun:     e.DryRun,
		Payload:    payloadBytes(e.Payload),
		OccurredAt: e.OccurredAt,
	})
	if err != nil {
		return nil, err
	}

	return eventFromPG(d), nil
}

func (p *TicketEventRepo) InsertBatch(ctx context.Context, events []*models.TicketEvent) error {
	if len(events) == 0 {
		return nil
	}

	rows := make([][]any, len(events))
	for i, e := range events {
		rows[i] = []any{e.TicketID, e.RunID, string(e.Kind), string(e.Source), e.DryRun, payloadBytes(e.Payload), e.OccurredAt}
	}

	src := pgxCopyRows(rows)
	_, err := p.copier.CopyFrom(
		ctx,
		pgx.Identifier{"ticket_event"},
		[]string{"ticket_id", "run_id", "kind", "source", "dry_run", "payload", "occurred_at"},
		&src,
	)

	return err
}

func (p *TicketEventRepo) ListByTicket(ctx context.Context, ticketID int, limit int, beforeID *int64) ([]*models.TicketEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	rows, err := p.queries.ListTicketEventsByTicket(ctx, db.ListTicketEventsByTicketParams{
		TicketID: ticketID,
		BeforeID: beforeID,
		Lim:      int32(limit),
	})
	if err != nil {
		return nil, err
	}

	return eventsFromPG(rows), nil
}

func (p *TicketEventRepo) ListRecent(ctx context.Context, kind *models.EventKind, limit int) ([]*models.TicketEvent, error) {
	if limit <= 0 {
		limit = 100
	}

	var k *string
	if kind != nil {
		s := string(*kind)
		k = &s
	}

	rows, err := p.queries.ListRecentTicketEvents(ctx, db.ListRecentTicketEventsParams{
		Kind: k,
		Lim:  int32(limit),
	})
	if err != nil {
		return nil, err
	}

	return eventsFromPG(rows), nil
}

func (p *TicketEventRepo) ListByRun(ctx context.Context, runID string) ([]*models.TicketEvent, error) {
	rows, err := p.queries.ListTicketEventsByRun(ctx, runID)
	if err != nil {
		return nil, err
	}
	return eventsFromPG(rows), nil
}

func (p *TicketEventRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return p.queries.DeleteTicketEventsBefore(ctx, before)
}

func payloadBytes(p json.RawMessage) []byte {
	if len(p) == 0 {
		return []byte("{}")
	}
	return []byte(p)
}

func eventsFromPG(rows []*db.TicketEvent) []*models.TicketEvent {
	out := make([]*models.TicketEvent, 0, len(rows))
	for _, r := range rows {
		out = append(out, eventFromPG(r))
	}
	return out
}

func eventFromPG(d *db.TicketEvent) *models.TicketEvent {
	return &models.TicketEvent{
		ID:         d.ID,
		TicketID:   d.TicketID,
		RunID:      d.RunID,
		Kind:       models.EventKind(d.Kind),
		Source:     models.EventSource(d.Source),
		DryRun:     d.DryRun,
		Payload:    json.RawMessage(d.Payload),
		OccurredAt: d.OccurredAt,
	}
}
