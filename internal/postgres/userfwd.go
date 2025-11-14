package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type UserForwardRepo struct {
	queries *db.Queries
}

func NewUserForwardRepo(pool *pgxpool.Pool) *UserForwardRepo {
	return &UserForwardRepo{
		queries: db.New(pool),
	}
}

func (p *UserForwardRepo) WithTx(tx pgx.Tx) models.UserForwardRepository {
	return &UserForwardRepo{
		queries: db.New(tx)}
}

func (p *UserForwardRepo) ListAll(ctx context.Context) ([]models.UserForward, error) {
	dm, err := p.queries.ListWebexUserForwards(ctx)
	if err != nil {
		return nil, err
	}

	var b []models.UserForward
	for _, d := range dm {
		b = append(b, forwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListByEmail(ctx context.Context, email string) ([]models.UserForward, error) {
	dm, err := p.queries.ListWebexUserForwardsByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var b []models.UserForward
	for _, d := range dm {
		b = append(b, forwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) Get(ctx context.Context, id int) (models.UserForward, error) {
	d, err := p.queries.GetWebexUserForward(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.UserForward{}, models.ErrUserForwardNotFound
		}
		return models.UserForward{}, err
	}

	return forwardFromPG(d), nil
}

func (p *UserForwardRepo) Insert(ctx context.Context, b models.UserForward) (models.UserForward, error) {
	d, err := p.queries.InsertWebexUserForward(ctx, forwardToInsertParams(b))
	if err != nil {
		return models.UserForward{}, err
	}

	return forwardFromPG(d), nil
}

func (p *UserForwardRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteWebexForward(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrUserForwardNotFound
		}
		return err
	}

	return nil
}

func forwardToInsertParams(t models.UserForward) db.InsertWebexUserForwardParams {
	return db.InsertWebexUserForwardParams{
		UserEmail:     t.UserEmail,
		DestRoomID:    t.DestRoomID,
		StartDate:     t.StartDate,
		EndDate:       t.EndDate,
		Enabled:       t.Enabled,
		UserKeepsCopy: t.UserKeepsCopy,
	}
}

func forwardFromPG(pg db.WebexUserForward) models.UserForward {
	return models.UserForward{
		ID:            pg.ID,
		UserEmail:     pg.UserEmail,
		DestRoomID:    pg.DestRoomID,
		StartDate:     pg.StartDate,
		EndDate:       pg.EndDate,
		Enabled:       pg.Enabled,
		UserKeepsCopy: pg.UserKeepsCopy,
		UpdatedOn:     pg.UpdatedOn,
		AddedOn:       pg.CreatedOn,
	}
}
