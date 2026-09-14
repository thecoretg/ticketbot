package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
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

func (p *TicketRepo) ListPaged(ctx context.Context, f models.TicketFilter) ([]*models.TicketListItem, int, error) {
	var search *string
	if s := strings.TrimSpace(f.Search); s != "" {
		search = &s
	}

	includeDeleted := f.IncludeDeleted
	page := max(f.Page, 1)
	size := f.PageSize
	if size <= 0 {
		size = 50
	}

	rows, err := p.queries.ListTicketsPaged(ctx, db.ListTicketsPagedParams{
		BoardID:        f.BoardID,
		StatusID:       f.StatusID,
		Closed:         f.Closed,
		IncludeDeleted: &includeDeleted,
		Search:         search,
		Lim:            int32(size),
		Off:            int32((page - 1) * size),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := p.queries.CountTicketsPaged(ctx, db.CountTicketsPagedParams{
		BoardID:        f.BoardID,
		StatusID:       f.StatusID,
		Closed:         f.Closed,
		IncludeDeleted: &includeDeleted,
		Search:         search,
	})
	if err != nil {
		return nil, 0, err
	}

	items := make([]*models.TicketListItem, 0, len(rows))
	for _, r := range rows {
		items = append(items, ticketListItemFromPG(r))
	}

	return items, int(total), nil
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
	var raw []byte
	if len(t.Raw) > 0 {
		raw = []byte(t.Raw)
	}

	return db.UpsertTicketParams{
		ID:           t.ID,
		Summary:      t.Summary,
		BoardID:      t.BoardID,
		StatusID:     t.StatusID,
		OwnerID:      t.OwnerID,
		CompanyID:    t.CompanyID,
		ContactID:    t.ContactID,
		Resources:    t.Resources,
		UpdatedBy:    t.UpdatedBy,
		PriorityID:   t.PriorityID,
		PriorityName: t.PriorityName,
		TypeID:       t.TypeID,
		TypeName:     t.TypeName,
		SubtypeID:    t.SubTypeID,
		SubtypeName:  t.SubTypeName,
		ItemID:       t.ItemID,
		ItemName:     t.ItemName,
		ClosedFlag:   t.ClosedFlag,
		LatestNoteID: t.LatestNoteID,
		Raw:          raw,
	}
}

func ticketFromPG(pg *db.CwTicket) *models.Ticket {
	return &models.Ticket{
		ID:           pg.ID,
		Summary:      pg.Summary,
		BoardID:      pg.BoardID,
		StatusID:     pg.StatusID,
		OwnerID:      pg.OwnerID,
		CompanyID:    pg.CompanyID,
		ContactID:    pg.ContactID,
		Resources:    pg.Resources,
		UpdatedBy:    pg.UpdatedBy,
		PriorityID:   pg.PriorityID,
		PriorityName: pg.PriorityName,
		TypeID:       pg.TypeID,
		TypeName:     pg.TypeName,
		SubTypeID:    pg.SubtypeID,
		SubTypeName:  pg.SubtypeName,
		ItemID:       pg.ItemID,
		ItemName:     pg.ItemName,
		ClosedFlag:   pg.ClosedFlag,
		LatestNoteID: pg.LatestNoteID,
		UpdatedOn:    pg.UpdatedOn,
		AddedOn:      pg.AddedOn,
		Deleted:      pg.Deleted,
		Raw:          pg.Raw,
	}
}

func ticketListItemFromPG(r *db.ListTicketsPagedRow) *models.TicketListItem {
	owner := ""
	if r.OwnerFirstName != nil || r.OwnerLastName != nil {
		owner = strings.TrimSpace(derefStr(r.OwnerFirstName) + " " + derefStr(r.OwnerLastName))
	}

	return &models.TicketListItem{
		Ticket: models.Ticket{
			ID:           r.ID,
			Summary:      r.Summary,
			BoardID:      r.BoardID,
			StatusID:     r.StatusID,
			OwnerID:      r.OwnerID,
			CompanyID:    r.CompanyID,
			ContactID:    r.ContactID,
			Resources:    r.Resources,
			UpdatedBy:    r.UpdatedBy,
			PriorityID:   r.PriorityID,
			PriorityName: r.PriorityName,
			TypeID:       r.TypeID,
			TypeName:     r.TypeName,
			SubTypeID:    r.SubtypeID,
			SubTypeName:  r.SubtypeName,
			ItemID:       r.ItemID,
			ItemName:     r.ItemName,
			ClosedFlag:   r.ClosedFlag,
			LatestNoteID: r.LatestNoteID,
			UpdatedOn:    r.UpdatedOn,
			AddedOn:      r.AddedOn,
			Deleted:      r.Deleted,
		},
		BoardName:   r.BoardName,
		StatusName:  r.StatusName,
		CompanyName: r.CompanyName,
		OwnerName:   owner,
	}
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
