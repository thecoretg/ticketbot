package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type TicketItemRepo struct {
	queries *db.Queries
}

func NewTicketItemRepo(pool *pgxpool.Pool) *TicketItemRepo {
	return &TicketItemRepo{
		queries: db.New(pool),
	}
}

func (p *TicketItemRepo) WithTx(tx pgx.Tx) repos.TicketItemRepository {
	return &TicketItemRepo{
		queries: db.New(tx),
	}
}

func (p *TicketItemRepo) List(ctx context.Context) ([]*models.TicketItem, error) {
	dbs, err := p.queries.ListAllTicketItems(ctx)
	if err != nil {
		return nil, err
	}

	var items []*models.TicketItem
	for _, d := range dbs {
		items = append(items, ticketItemFromPG(d))
	}

	return items, nil
}

func (p *TicketItemRepo) ListByBoard(ctx context.Context, boardID int) ([]*models.TicketItem, error) {
	dbs, err := p.queries.ListTicketItemsByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var items []*models.TicketItem
	for _, d := range dbs {
		items = append(items, ticketItemFromPG(d))
	}

	return items, nil
}

func (p *TicketItemRepo) Get(ctx context.Context, id int) (*models.TicketItem, error) {
	d, err := p.queries.GetTicketItem(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketItemNotFound
		}
		return nil, err
	}

	return ticketItemFromPG(d), nil
}

func (p *TicketItemRepo) Upsert(ctx context.Context, t *models.TicketItem) (*models.TicketItem, error) {
	d, err := p.queries.UpsertTicketItem(ctx, ticketItemToUpsertParams(t))
	if err != nil {
		return nil, err
	}

	return ticketItemFromPG(d), nil
}

func (p *TicketItemRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteTicketItem(ctx, id)
}

func (p *TicketItemRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketItem(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketItemNotFound
		}
		return err
	}

	return nil
}

func ticketItemToUpsertParams(t *models.TicketItem) db.UpsertTicketItemParams {
	return db.UpsertTicketItemParams{
		ID:       t.ID,
		BoardID:  t.BoardID,
		Name:     t.Name,
		Inactive: t.Inactive,
	}
}

func ticketItemFromPG(pg *db.CwTicketItem) *models.TicketItem {
	return &models.TicketItem{
		ID:        pg.ID,
		BoardID:   pg.BoardID,
		Name:      pg.Name,
		Inactive:  pg.Inactive,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
