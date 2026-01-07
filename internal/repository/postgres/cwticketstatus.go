package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type TicketStatusRepo struct {
	queries *db.Queries
}

func NewTicketStatusRepo(pool *pgxpool.Pool) *TicketStatusRepo {
	return &TicketStatusRepo{
		queries: db.New(pool),
	}
}

func (p *TicketStatusRepo) WithTx(tx pgx.Tx) models.TicketStatusRepository {
	return &TicketStatusRepo{
		queries: db.New(tx),
	}
}

func (p *TicketStatusRepo) List(ctx context.Context) ([]*models.TicketStatus, error) {
	dbs, err := p.queries.ListAllTicketStatuses(ctx)
	if err != nil {
		return nil, err
	}

	var statuses []*models.TicketStatus
	for _, d := range dbs {
		statuses = append(statuses, ticketStatusFromPG(d))
	}

	return statuses, nil
}

func (p *TicketStatusRepo) ListByBoard(ctx context.Context, boardID int) ([]*models.TicketStatus, error) {
	dbs, err := p.queries.ListTicketStatusesByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var statuses []*models.TicketStatus
	for _, d := range dbs {
		statuses = append(statuses, ticketStatusFromPG(d))
	}

	return statuses, nil
}

func (p *TicketStatusRepo) Get(ctx context.Context, id int) (*models.TicketStatus, error) {
	d, err := p.queries.GetTicketStatus(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTicketStatusNotFound
		}
		return nil, err
	}

	return ticketStatusFromPG(d), nil
}

func (p *TicketStatusRepo) Upsert(ctx context.Context, s *models.TicketStatus) (*models.TicketStatus, error) {
	d, err := p.queries.UpsertTicketStatus(ctx, ticketStatusToUpsertParams(s))
	if err != nil {
		return nil, err
	}

	return ticketStatusFromPG(d), nil
}

func (p *TicketStatusRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTicketStatus(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTicketStatusNotFound
		}
		return err
	}

	return nil
}

func ticketStatusToUpsertParams(s *models.TicketStatus) db.UpsertTicketStatusParams {
	return db.UpsertTicketStatusParams{
		ID:             s.ID,
		BoardID:        s.BoardID,
		Name:           s.Name,
		DefaultStatus:  s.DefaultStatus,
		DisplayOnBoard: s.DisplayOnBoard,
		Inactive:       s.Inactive,
		Closed:         s.Closed,
	}
}

func ticketStatusFromPG(pg *db.CwTicketStatus) *models.TicketStatus {
	return &models.TicketStatus{
		ID:             pg.ID,
		BoardID:        pg.BoardID,
		Name:           pg.Name,
		DefaultStatus:  pg.DefaultStatus,
		DisplayOnBoard: pg.DisplayOnBoard,
		Inactive:       pg.Inactive,
		Closed:         pg.Closed,
		UpdatedOn:      pg.UpdatedOn,
		AddedOn:        pg.AddedOn,
	}
}
