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

type TicketTypeRepo struct {
	queries *db.Queries
}

func NewTicketTypeRepo(pool *pgxpool.Pool) *TicketTypeRepo {
	return &TicketTypeRepo{
		queries: db.New(pool),
	}
}

func (p *TicketTypeRepo) WithTx(tx pgx.Tx) repos.TicketTypeRepository {
	return &TicketTypeRepo{
		queries: db.New(tx),
	}
}

func (p *TicketTypeRepo) List(ctx context.Context) ([]*models.TicketType, error) {
	dbs, err := p.queries.ListAllTicketTypes(ctx)
	if err != nil {
		return nil, err
	}

	var types []*models.TicketType
	for _, d := range dbs {
		types = append(types, ticketTypeFromPG(d))
	}

	return types, nil
}

func (p *TicketTypeRepo) ListByBoard(ctx context.Context, boardID int) ([]*models.TicketType, error) {
	dbs, err := p.queries.ListTicketTypesByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var types []*models.TicketType
	for _, d := range dbs {
		types = append(types, ticketTypeFromPG(d))
	}

	return types, nil
}

func (p *TicketTypeRepo) Get(ctx context.Context, id int) (*models.TicketType, error) {
	d, err := p.queries.GetTicketType(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketTypeNotFound
		}
		return nil, err
	}

	return ticketTypeFromPG(d), nil
}

func (p *TicketTypeRepo) Upsert(ctx context.Context, t *models.TicketType) (*models.TicketType, error) {
	d, err := p.queries.UpsertTicketType(ctx, ticketTypeToUpsertParams(t))
	if err != nil {
		return nil, err
	}

	return ticketTypeFromPG(d), nil
}

func (p *TicketTypeRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteTicketType(ctx, id)
}

func (p *TicketTypeRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketType(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketTypeNotFound
		}
		return err
	}

	return nil
}

func ticketTypeToUpsertParams(t *models.TicketType) db.UpsertTicketTypeParams {
	return db.UpsertTicketTypeParams{
		ID:          t.ID,
		BoardID:     t.BoardID,
		Name:        t.Name,
		Category:    t.Category,
		DefaultFlag: t.DefaultFlag,
		Inactive:    t.Inactive,
	}
}

func ticketTypeFromPG(pg *db.CwTicketType) *models.TicketType {
	return &models.TicketType{
		ID:          pg.ID,
		BoardID:     pg.BoardID,
		Name:        pg.Name,
		Category:    pg.Category,
		DefaultFlag: pg.DefaultFlag,
		Inactive:    pg.Inactive,
		UpdatedOn:   pg.UpdatedOn,
		AddedOn:     pg.AddedOn,
		Deleted:     pg.Deleted,
	}
}
