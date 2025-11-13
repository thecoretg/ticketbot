package note

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
)

type PostgresRepo struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (p *PostgresRepo) ListByTicketID(ctx context.Context, ticketID int) ([]TicketNote, error) {
	dm, err := p.queries.ListTicketNotesByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	var b []TicketNote
	for _, d := range dm {
		b = append(b, ticketNoteFromPG(d))
	}

	return b, nil
}

func (p *PostgresRepo) ListAll(ctx context.Context) ([]TicketNote, error) {
	dm, err := p.queries.ListAllTicketNotes(ctx)
	if err != nil {
		return nil, err
	}

	var b []TicketNote
	for _, d := range dm {
		b = append(b, ticketNoteFromPG(d))
	}

	return b, nil
}

func (p *PostgresRepo) Get(ctx context.Context, id int) (TicketNote, error) {
	d, err := p.queries.GetTicketNote(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return TicketNote{}, ErrNotFound
		}
		return TicketNote{}, err
	}

	return ticketNoteFromPG(d), nil
}

func (p *PostgresRepo) Upsert(ctx context.Context, b TicketNote) (TicketNote, error) {
	d, err := p.queries.UpsertTicketNote(ctx, pgUpsertParams(b))
	if err != nil {
		return TicketNote{}, err
	}

	return ticketNoteFromPG(d), nil
}

func (p *PostgresRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicket(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}

	return nil
}

func pgUpsertParams(t TicketNote) db.UpsertTicketNoteParams {
	return db.UpsertTicketNoteParams{
		ID:            t.ID,
		TicketID:      t.TicketID,
		MemberID:      t.MemberID,
		ContactID:     t.ContactID,
		Notified:      t.Notified,
		SkippedNotify: t.SkippedNotify,
	}
}

func ticketNoteFromPG(pg db.CwTicketNote) TicketNote {
	return TicketNote{
		ID:            pg.ID,
		TicketID:      pg.TicketID,
		MemberID:      pg.MemberID,
		ContactID:     pg.ContactID,
		Notified:      pg.Notified,
		SkippedNotify: pg.SkippedNotify,
		UpdatedOn:     pg.UpdatedOn,
		AddedOn:       pg.AddedOn,
	}
}
