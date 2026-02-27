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

type TicketRepo struct {
	queries *db.Queries
}

func NewTicketRepo(pool *pgxpool.Pool) *TicketRepo {
	return &TicketRepo{
		queries: db.New(pool),
	}
}

func (p *TicketRepo) WithTx(tx pgx.Tx) repos.TicketRepository {
	return &TicketRepo{
		queries: db.New(tx),
	}
}

func (p *TicketRepo) List(ctx context.Context) ([]*models.Ticket, error) {
	dm, err := p.queries.ListTickets(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.Ticket
	for _, d := range dm {
		b = append(b, ticketFromPG(d))
	}

	return b, nil
}

func (p *TicketRepo) Get(ctx context.Context, id int) (*models.Ticket, error) {
	d, err := p.queries.GetTicket(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketNotFound
		}
		return nil, err
	}

	return ticketFromPG(d), nil
}

func (p *TicketRepo) Exists(ctx context.Context, id int) (bool, error) {
	return p.queries.CheckTicketExists(ctx, id)
}

func (p *TicketRepo) Upsert(ctx context.Context, b *models.Ticket) (*models.Ticket, error) {
	d, err := p.queries.UpsertTicket(ctx, ticketToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return ticketFromPG(d), nil
}

func (p *TicketRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteTicket(ctx, id)
}

func (p *TicketRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicket(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketNotFound
		}
		return err
	}

	return nil
}

func ticketToUpsertParams(t *models.Ticket) db.UpsertTicketParams {
	return db.UpsertTicketParams{
		ID:        t.ID,
		Summary:   t.Summary,
		BoardID:   t.BoardID,
		StatusID:  t.StatusID,
		OwnerID:   t.OwnerID,
		CompanyID: t.CompanyID,
		ContactID: t.ContactID,
		Resources: t.Resources,
		UpdatedBy: t.UpdatedBy,
	}
}

func ticketFromPG(pg *db.CwTicket) *models.Ticket {
	return &models.Ticket{
		ID:        pg.ID,
		Summary:   pg.Summary,
		BoardID:   pg.BoardID,
		StatusID:  pg.StatusID,
		OwnerID:   pg.OwnerID,
		CompanyID: pg.CompanyID,
		ContactID: pg.ContactID,
		Resources: pg.Resources,
		UpdatedBy: pg.UpdatedBy,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
