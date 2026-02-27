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

type UserForwardRepo struct {
	queries *db.Queries
}

func NewUserForwardRepo(pool *pgxpool.Pool) *UserForwardRepo {
	return &UserForwardRepo{
		queries: db.New(pool),
	}
}

func (p *UserForwardRepo) WithTx(tx pgx.Tx) repos.NotifierForwardRepository {
	return &UserForwardRepo{
		queries: db.New(tx),
	}
}

func (p *UserForwardRepo) ListAll(ctx context.Context) ([]*models.NotifierForward, error) {
	dm, err := p.queries.ListNotifierForwards(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForward
	for _, d := range dm {
		b = append(b, forwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListAllActive(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	dm, err := p.queries.ListActiveNotifierForwards(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForwardFull
	for _, d := range dm {
		b = append(b, activeForwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListAllNotExpired(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	dm, err := p.queries.ListNotExpiredNotifierForwards(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForwardFull
	for _, d := range dm {
		b = append(b, notExpiredForwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListAllInactive(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	dm, err := p.queries.ListInactiveNotifierForwards(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForwardFull
	for _, d := range dm {
		b = append(b, inactiveForwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListAllFull(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	df, err := p.queries.ListNotifierForwardsFull(ctx)
	if err != nil {
		return nil, err
	}

	var f []*models.NotifierForwardFull
	for _, d := range df {
		f = append(f, fullForwardFromPG(d))
	}

	return f, nil
}

func (p *UserForwardRepo) ListBySourceRoomID(ctx context.Context, id int) ([]*models.NotifierForward, error) {
	dm, err := p.queries.ListNotifierForwardsBySourceRecipientID(ctx, id)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForward
	for _, d := range dm {
		b = append(b, forwardFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) ListActiveBySourceRoomID(ctx context.Context, id int) ([]*models.NotifierForwardFull, error) {
	dm, err := p.queries.ListActiveForwardsBySourceRecipient(ctx, id)
	if err != nil {
		return nil, err
	}

	var b []*models.NotifierForwardFull
	for _, d := range dm {
		b = append(b, activeForwardBySourceFromPG(d))
	}

	return b, nil
}

func (p *UserForwardRepo) Get(ctx context.Context, id int) (*models.NotifierForward, error) {
	d, err := p.queries.GetNotifierForward(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrUserForwardNotFound
		}
		return nil, err
	}

	return forwardFromPG(d), nil
}

func (p *UserForwardRepo) Exists(ctx context.Context, id int) (bool, error) {
	return p.queries.CheckNotifierForwardExists(ctx, id)
}

func (p *UserForwardRepo) Insert(ctx context.Context, b *models.NotifierForward) (*models.NotifierForward, error) {
	d, err := p.queries.InsertNotifierForward(ctx, forwardToInsertParams(b))
	if err != nil {
		return nil, err
	}

	return forwardFromPG(d), nil
}

func (p *UserForwardRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteNotifierForward(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrUserForwardNotFound
		}
		return err
	}

	return nil
}

func forwardToInsertParams(t *models.NotifierForward) db.InsertNotifierForwardParams {
	return db.InsertNotifierForwardParams{
		SourceID:      t.SourceID,
		DestinationID: t.DestID,
		StartDate:     t.StartDate,
		EndDate:       t.EndDate,
		Enabled:       t.Enabled,
		UserKeepsCopy: t.UserKeepsCopy,
	}
}

func forwardFromPG(pg *db.NotifierForward) *models.NotifierForward {
	return &models.NotifierForward{
		ID:            pg.ID,
		SourceID:      pg.SourceID,
		DestID:        pg.DestinationID,
		StartDate:     pg.StartDate,
		EndDate:       pg.EndDate,
		Enabled:       pg.Enabled,
		UserKeepsCopy: pg.UserKeepsCopy,
		UpdatedOn:     pg.UpdatedOn,
		CreatedOn:     pg.CreatedOn,
	}
}

func fullForwardFromPG(pg *db.ListNotifierForwardsFullRow) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		ID:              pg.ID,
		Enabled:         pg.Enabled,
		UserKeepsCopy:   pg.UserKeepsCopy,
		StartDate:       pg.StartDate,
		EndDate:         pg.EndDate,
		SourceID:        pg.SourceID,
		SourceName:      pg.SourceName,
		SourceType:      pg.SourceType,
		DestinationID:   pg.DestinationID,
		DestinationName: pg.DestinationName,
		DestinationType: pg.DestinationType,
	}
}

func activeForwardFromPG(pg *db.ListActiveNotifierForwardsRow) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		ID:              pg.ID,
		Enabled:         pg.Enabled,
		UserKeepsCopy:   pg.UserKeepsCopy,
		StartDate:       pg.StartDate,
		EndDate:         pg.EndDate,
		SourceID:        pg.SourceID,
		SourceName:      pg.SourceName,
		SourceType:      pg.SourceType,
		DestinationID:   pg.DestinationID,
		DestinationName: pg.DestinationName,
		DestinationType: pg.DestinationType,
	}
}

func inactiveForwardFromPG(pg *db.ListInactiveNotifierForwardsRow) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		ID:              pg.ID,
		Enabled:         pg.Enabled,
		UserKeepsCopy:   pg.UserKeepsCopy,
		StartDate:       pg.StartDate,
		EndDate:         pg.EndDate,
		SourceID:        pg.SourceID,
		SourceName:      pg.SourceName,
		SourceType:      pg.SourceType,
		DestinationID:   pg.DestinationID,
		DestinationName: pg.DestinationName,
		DestinationType: pg.DestinationType,
	}
}

func notExpiredForwardFromPG(pg *db.ListNotExpiredNotifierForwardsRow) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		ID:              pg.ID,
		Enabled:         pg.Enabled,
		UserKeepsCopy:   pg.UserKeepsCopy,
		StartDate:       pg.StartDate,
		EndDate:         pg.EndDate,
		SourceID:        pg.SourceID,
		SourceName:      pg.SourceName,
		SourceType:      pg.SourceType,
		DestinationID:   pg.DestinationID,
		DestinationName: pg.DestinationName,
		DestinationType: pg.DestinationType,
	}
}

func activeForwardBySourceFromPG(pg *db.ListActiveForwardsBySourceRecipientRow) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		ID:              pg.ID,
		Enabled:         pg.Enabled,
		UserKeepsCopy:   pg.UserKeepsCopy,
		StartDate:       pg.StartDate,
		EndDate:         pg.EndDate,
		SourceID:        pg.SourceID,
		SourceName:      pg.SourceName,
		SourceType:      pg.SourceType,
		DestinationID:   pg.DestinationID,
		DestinationName: pg.DestinationName,
		DestinationType: pg.DestinationType,
	}
}
