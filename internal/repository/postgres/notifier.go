package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotifierRuleRepo struct {
	queries *db.Queries
}

func NewNotifierRuleRepo(pool *pgxpool.Pool) *NotifierRuleRepo {
	return &NotifierRuleRepo{
		queries: db.New(pool),
	}
}

func (p *NotifierRuleRepo) WithTx(tx pgx.Tx) models.NotifierRuleRepository {
	return &NotifierRuleRepo{
		queries: db.New(tx),
	}
}

func (p *NotifierRuleRepo) ListAll(ctx context.Context) ([]models.NotifierRule, error) {
	dm, err := p.queries.ListNotifierRules(ctx)
	if err != nil {
		return nil, err
	}

	var b []models.NotifierRule
	for _, d := range dm {
		n := notifierFromPG(d)
		b = append(b, *n)
	}

	return b, nil
}

func (p *NotifierRuleRepo) ListByBoard(ctx context.Context, boardID int) ([]models.NotifierRule, error) {
	dm, err := p.queries.ListNotifierRulesByBoard(ctx, boardID)
	if err != nil {
		return nil, err
	}

	var b []models.NotifierRule
	for _, d := range dm {
		n := notifierFromPG(d)
		b = append(b, *n)
	}

	return b, nil
}

func (p *NotifierRuleRepo) ListByRoom(ctx context.Context, roomID int) ([]models.NotifierRule, error) {
	dm, err := p.queries.ListNotifierRulesByRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	var b []models.NotifierRule
	for _, d := range dm {
		n := notifierFromPG(d)
		b = append(b, *n)
	}

	return b, nil
}

func (p *NotifierRuleRepo) Get(ctx context.Context, id int) (*models.NotifierRule, error) {
	d, err := p.queries.GetNotifierRule(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotifierNotFound
		}
		return nil, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRuleRepo) Exists(ctx context.Context, boardID, roomID int) (bool, error) {
	ids := db.CheckNotifierExistsParams{
		CwBoardID:        boardID,
		WebexRecipientID: roomID,
	}

	exists, err := p.queries.CheckNotifierExists(ctx, ids)
	if err != nil {
		return false, err
	}

	return exists, nil
}

func (p *NotifierRuleRepo) Insert(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error) {
	d, err := p.queries.InsertNotifierRule(ctx, notifierToInsertParams(n))
	if err != nil {
		return nil, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRuleRepo) Update(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error) {
	d, err := p.queries.UpdateNotifierRule(ctx, notifierToUpdateParams(n))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrNotifierNotFound
		}
		return nil, err
	}

	return notifierFromPG(d), nil
}

func (p *NotifierRuleRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteNotifierRule(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrNotifierNotFound
		}
		return err
	}

	return nil
}

func notifierToInsertParams(n *models.NotifierRule) db.InsertNotifierRuleParams {
	return db.InsertNotifierRuleParams{
		CwBoardID:        n.CwBoardID,
		WebexRecipientID: n.WebexRoomID,
		NotifyEnabled:    n.NotifyEnabled,
	}
}

func notifierToUpdateParams(n *models.NotifierRule) db.UpdateNotifierRuleParams {
	return db.UpdateNotifierRuleParams{
		ID:               n.ID,
		CwBoardID:        n.CwBoardID,
		WebexRecipientID: n.WebexRoomID,
		NotifyEnabled:    n.NotifyEnabled,
	}
}

func notifierFromPG(pg db.NotifierRule) *models.NotifierRule {
	return &models.NotifierRule{
		ID:            pg.ID,
		CwBoardID:     pg.CwBoardID,
		WebexRoomID:   pg.WebexRecipientID,
		NotifyEnabled: pg.NotifyEnabled,
		CreatedOn:     pg.CreatedOn,
	}
}
