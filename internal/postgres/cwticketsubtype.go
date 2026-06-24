package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type TicketSubTypeRepo struct {
	queries *db.Queries
}

func NewTicketSubTypeRepo(pool *pgxpool.Pool) *TicketSubTypeRepo {
	return &TicketSubTypeRepo{
		queries: db.New(pool),
	}
}

func (p *TicketSubTypeRepo) WithTx(tx pgx.Tx) repos.TicketSubTypeRepository {
	return &TicketSubTypeRepo{
		queries: db.New(tx),
	}
}

func (p *TicketSubTypeRepo) List(ctx context.Context) ([]*models.TicketSubType, error) {
	dbs, err := p.queries.ListAllTicketSubTypes(ctx)
	if err != nil {
		return nil, err
	}

	var subtypes []*models.TicketSubType
	for _, d := range dbs {
		subtypes = append(subtypes, ticketSubTypeFromPG(d))
	}

	return subtypes, nil
}

func (p *TicketSubTypeRepo) ListByBoard(ctx context.Context, boardID int) ([]*models.TicketSubType, error) {
	dbs, err := p.queries.ListTicketSubTypesByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var subtypes []*models.TicketSubType
	for _, d := range dbs {
		subtypes = append(subtypes, ticketSubTypeFromPG(d))
	}

	return subtypes, nil
}

func (p *TicketSubTypeRepo) Get(ctx context.Context, id int) (*models.TicketSubType, error) {
	d, err := p.queries.GetTicketSubType(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketSubTypeNotFound
		}
		return nil, err
	}

	return ticketSubTypeFromPG(d), nil
}

func (p *TicketSubTypeRepo) Upsert(ctx context.Context, t *models.TicketSubType) (*models.TicketSubType, error) {
	params, err := ticketSubTypeToUpsertParams(t)
	if err != nil {
		return nil, err
	}

	d, err := p.queries.UpsertTicketSubType(ctx, params)
	if err != nil {
		return nil, err
	}

	return ticketSubTypeFromPG(d), nil
}

func (p *TicketSubTypeRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteTicketSubType(ctx, id)
}

func (p *TicketSubTypeRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketSubType(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketSubTypeNotFound
		}
		return err
	}

	return nil
}

func ticketSubTypeToUpsertParams(t *models.TicketSubType) (db.UpsertTicketSubTypeParams, error) {
	ids := t.TypeAssociationIDs
	if ids == nil {
		ids = []int{}
	}
	raw, err := json.Marshal(ids)
	if err != nil {
		return db.UpsertTicketSubTypeParams{}, err
	}

	return db.UpsertTicketSubTypeParams{
		ID:                 t.ID,
		BoardID:            t.BoardID,
		Name:               t.Name,
		Inactive:           t.Inactive,
		TypeAssociationIds: raw,
	}, nil
}

func ticketSubTypeFromPG(pg *db.CwTicketSubtype) *models.TicketSubType {
	var ids []int
	if len(pg.TypeAssociationIds) > 0 {
		_ = json.Unmarshal(pg.TypeAssociationIds, &ids)
	}

	return &models.TicketSubType{
		ID:                 pg.ID,
		BoardID:            pg.BoardID,
		Name:               pg.Name,
		Inactive:           pg.Inactive,
		TypeAssociationIDs: ids,
		UpdatedOn:          pg.UpdatedOn,
		AddedOn:            pg.AddedOn,
		Deleted:            pg.Deleted,
	}
}
