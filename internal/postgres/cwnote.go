package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type TicketNoteRepo struct {
	queries *db.Queries
}

func NewTicketNoteRepo(pool *pgxpool.Pool) *TicketNoteRepo {
	return &TicketNoteRepo{
		queries: db.New(pool),
	}
}

func (p *TicketNoteRepo) WithTx(tx pgx.Tx) repos.TicketNoteRepository {
	return &TicketNoteRepo{
		queries: db.New(tx)}
}

func (p *TicketNoteRepo) ListByTicketID(ctx context.Context, ticketID int) ([]*models.TicketNote, error) {
	dm, err := p.queries.ListTicketNotesByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}

	var b []*models.TicketNote
	for _, d := range dm {
		b = append(b, ticketNoteFromPG(d))
	}

	return b, nil
}

func (p *TicketNoteRepo) ListAll(ctx context.Context) ([]*models.TicketNote, error) {
	dm, err := p.queries.ListAllTicketNotes(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.TicketNote
	for _, d := range dm {
		b = append(b, ticketNoteFromPG(d))
	}

	return b, nil
}

func (p *TicketNoteRepo) Get(ctx context.Context, id int) (*models.TicketNote, error) {
	d, err := p.queries.GetTicketNote(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketNoteNotFound
		}
		return nil, err
	}

	return ticketNoteFromPG(d), nil
}

func (p *TicketNoteRepo) Upsert(ctx context.Context, b *models.TicketNote) (*models.TicketNote, error) {
	d, err := p.queries.UpsertTicketNote(ctx, ticketNoteToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return ticketNoteFromPG(d), nil
}

func (p *TicketNoteRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteTicketNote(ctx, id)
}

func (p *TicketNoteRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketNote(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketNoteNotFound
		}
		return err
	}

	return nil
}

func ticketNoteToUpsertParams(t *models.TicketNote) db.UpsertTicketNoteParams {
	return db.UpsertTicketNoteParams{
		ID:        t.ID,
		TicketID:  t.TicketID,
		Content:   t.Content,
		MemberID:  t.MemberID,
		ContactID: t.ContactID,
	}
}

func ticketNoteFromPG(pg *db.CwTicketNote) *models.TicketNote {
	return &models.TicketNote{
		ID:        pg.ID,
		TicketID:  pg.TicketID,
		Content:   pg.Content,
		MemberID:  pg.MemberID,
		ContactID: pg.ContactID,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
