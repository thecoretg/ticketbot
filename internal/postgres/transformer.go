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

type TransformerRuleRepo struct {
	queries *db.Queries
}

func NewTransformerRuleRepo(pool *pgxpool.Pool) *TransformerRuleRepo {
	return &TransformerRuleRepo{
		queries: db.New(pool),
	}
}

func (p *TransformerRuleRepo) WithTx(tx pgx.Tx) repos.TransformerRuleRepository {
	return &TransformerRuleRepo{
		queries: db.New(tx),
	}
}

func (p *TransformerRuleRepo) ListAll(ctx context.Context) ([]*models.TransformerRule, error) {
	dm, err := p.queries.ListTransformerRules(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.TransformerRule
	for _, d := range dm {
		b = append(b, transformerRuleFromPG(d))
	}

	return b, nil
}

func (p *TransformerRuleRepo) ListEnabled(ctx context.Context) ([]*models.TransformerRule, error) {
	dm, err := p.queries.ListEnabledTransformerRules(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.TransformerRule
	for _, d := range dm {
		b = append(b, transformerRuleFromPG(d))
	}

	return b, nil
}

func (p *TransformerRuleRepo) Get(ctx context.Context, id int) (*models.TransformerRule, error) {
	d, err := p.queries.GetTransformerRule(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTransformerRuleNotFound
		}
		return nil, err
	}

	return transformerRuleFromPG(d), nil
}

func (p *TransformerRuleRepo) Exists(ctx context.Context, id int) (bool, error) {
	return p.queries.CheckTransformerRuleExists(ctx, id)
}

func (p *TransformerRuleRepo) Insert(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error) {
	d, err := p.queries.InsertTransformerRule(ctx, transformerRuleToInsertParams(r))
	if err != nil {
		return nil, err
	}

	return transformerRuleFromPG(d), nil
}

func (p *TransformerRuleRepo) Update(ctx context.Context, r *models.TransformerRule) (*models.TransformerRule, error) {
	d, err := p.queries.UpdateTransformerRule(ctx, transformerRuleToUpdateParams(r))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrTransformerRuleNotFound
		}
		return nil, err
	}

	return transformerRuleFromPG(d), nil
}

func (p *TransformerRuleRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteTransformerRule(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrTransformerRuleNotFound
		}
		return err
	}

	return nil
}

func transformerRuleToInsertParams(r *models.TransformerRule) db.InsertTransformerRuleParams {
	return db.InsertTransformerRuleParams{
		Name:       r.Name,
		Action:     r.Action,
		CwBoardID:  r.CwBoardID,
		Config:     configBytes(r.Config),
		Conditions: conditionsBytes(r.Conditions),
		ApplyOn:    r.ApplyOn,
		Priority:   r.Priority,
		Enabled:    r.Enabled,
	}
}

func transformerRuleToUpdateParams(r *models.TransformerRule) db.UpdateTransformerRuleParams {
	return db.UpdateTransformerRuleParams{
		ID:         r.ID,
		Name:       r.Name,
		Action:     r.Action,
		CwBoardID:  r.CwBoardID,
		Config:     configBytes(r.Config),
		Conditions: conditionsBytes(r.Conditions),
		ApplyOn:    r.ApplyOn,
		Priority:   r.Priority,
		Enabled:    r.Enabled,
	}
}

func transformerRuleFromPG(pg *db.TransformerRule) *models.TransformerRule {
	return &models.TransformerRule{
		ID:         pg.ID,
		Name:       pg.Name,
		Action:     pg.Action,
		CwBoardID:  pg.CwBoardID,
		Config:     pg.Config,
		Conditions: conditionsFromBytes(pg.Conditions),
		ApplyOn:    pg.ApplyOn,
		Priority:   pg.Priority,
		Enabled:    pg.Enabled,
		CreatedOn:  pg.CreatedOn,
		UpdatedOn:  pg.UpdatedOn,
	}
}

// configBytes ensures a non-nil JSON object is stored for the NOT NULL jsonb column.
func configBytes(c []byte) []byte {
	if len(c) == 0 {
		return []byte("{}")
	}
	return c
}

// conditionsBytes marshals rule conditions to the NOT NULL jsonb column, storing
// an empty array when there are none.
func conditionsBytes(conds []models.RuleCondition) []byte {
	if len(conds) == 0 {
		return []byte("[]")
	}
	b, err := json.Marshal(conds)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func conditionsFromBytes(b []byte) []models.RuleCondition {
	if len(b) == 0 {
		return nil
	}
	var conds []models.RuleCondition
	if err := json.Unmarshal(b, &conds); err != nil {
		return nil
	}
	return conds
}

type TransformerRunRepo struct {
	queries *db.Queries
}

func NewTransformerRunRepo(pool *pgxpool.Pool) *TransformerRunRepo {
	return &TransformerRunRepo{
		queries: db.New(pool),
	}
}

func (p *TransformerRunRepo) WithTx(tx pgx.Tx) repos.TransformerRunRepository {
	return &TransformerRunRepo{
		queries: db.New(tx),
	}
}

func (p *TransformerRunRepo) Exists(ctx context.Context, ticketID, ruleID int) (bool, error) {
	return p.queries.CheckTransformerRunExists(ctx, db.CheckTransformerRunExistsParams{
		TicketID: ticketID,
		RuleID:   ruleID,
	})
}

func (p *TransformerRunRepo) Insert(ctx context.Context, ticketID, ruleID int) error {
	return p.queries.InsertTransformerRun(ctx, db.InsertTransformerRunParams{
		TicketID: ticketID,
		RuleID:   ruleID,
	})
}
