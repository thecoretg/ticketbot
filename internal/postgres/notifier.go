package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotifierRepo struct {
	queries *db.Queries
}

func NewNotifierRepo(pool *pgxpool.Pool) *NotifierRepo {
	return &NotifierRepo{
		queries: db.New(pool),
	}
}

func (p *NotifierRepo) WithTx(tx pgx.Tx) models.NotifierRepository {
	return &NotifierRepo{
		queries: db.New(tx)}
}

func (p *NotifierRepo) ListAll(ctx context.Context) ([]models.Notifier, error) {
	dm, err := p.queries.ListNotifierConnections(ctx)
	if err != nil {
		return nil, err
	}

	var b []models.Notifier
	for _, d := range dm {
		b = append(b, notifierFromPG(d))
	}

	return b, nil
}

func (p *NotifierRepo) ListByBoard(ctx context.Context, boardID int) ([]models.Notifier, error) {
	dm, err := p.queries.ListNotifierConnectionsByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var b []models.Notifier
	for _, d := range dm {
		b = append(b, notifierFromPG(d))
	}

	return b, nil
}

func (p *NotifierRepo) ListByRoom(ctx context.Context, roomID int) ([]models.Notifier, error) {
	dm, err := p.queries.ListNotifierConnectionsByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	var b []models.Notifier
	for _, d := range dm {
		b = append(b, notifierFromPG(d))
	}

	return b, nil
}

func (p *NotifierRepo) Get(ctx context.Context, id int) (models.Notifier, error) {
	d, err := p.queries.GetNotifierConnection(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Notifier{}, models.ErrNotifierNotFound
		}
		return models.Notifier{}, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRepo) Insert(ctx context.Context, n models.Notifier) (models.Notifier, error) {
	d, err := p.queries.InsertNotifierConnection(ctx, notifierToInsertParams(n))
	if err != nil {
		return models.Notifier{}, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRepo) Update(ctx context.Context, n models.Notifier) (models.Notifier, error) {
	d, err := p.queries.UpdateNotifierConnection(ctx, notifierToUpdateParams(n))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Notifier{}, models.ErrNotifierNotFound
		}
		return models.Notifier{}, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteNotifierConnection(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrNotifierNotFound
		}
		return err
	}

	return nil
}

func notifierToInsertParams(n models.Notifier) db.InsertNotifierConnectionParams {
	return db.InsertNotifierConnectionParams{
		CwBoardID:     n.CwBoardID,
		WebexRoomID:   n.WebexRoomID,
		NotifyEnabled: n.NotifyEnabled,
	}
}

func notifierToUpdateParams(n models.Notifier) db.UpdateNotifierConnectionParams {
	return db.UpdateNotifierConnectionParams{
		ID:            n.ID,
		CwBoardID:     n.CwBoardID,
		WebexRoomID:   n.WebexRoomID,
		NotifyEnabled: n.NotifyEnabled,
	}
}

func notifierFromPG(pg db.NotifierConnection) models.Notifier {
	return models.Notifier{
		ID:            pg.ID,
		CwBoardID:     pg.CwBoardID,
		WebexRoomID:   pg.WebexRoomID,
		NotifyEnabled: pg.NotifyEnabled,
		CreatedOn:     pg.CreatedOn,
	}
}
