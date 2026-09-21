package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowRepo struct {
	queries *db.Queries
}

func NewWorkflowRepo(pool *pgxpool.Pool) *WorkflowRepo {
	return &WorkflowRepo{queries: db.New(pool)}
}

func (p *WorkflowRepo) WithTx(tx pgx.Tx) repos.WorkflowRepository {
	return &WorkflowRepo{queries: db.New(tx)}
}

func (p *WorkflowRepo) List(ctx context.Context) ([]*models.Workflow, error) {
	rows, err := p.queries.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]*models.Workflow, 0, len(rows))
	for _, r := range rows {
		w, err := workflowFromParts(r.ID, r.BoardID, r.BoardName, r.Name, r.Enabled, r.DryRun, r.Rules, r.CreatedOn, r.UpdatedOn)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}

	return out, nil
}

func (p *WorkflowRepo) Get(ctx context.Context, id int) (*models.Workflow, error) {
	r, err := p.queries.GetWorkflow(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWorkflowNotFound
		}
		return nil, err
	}

	return workflowFromParts(r.ID, r.BoardID, r.BoardName, r.Name, r.Enabled, r.DryRun, r.Rules, r.CreatedOn, r.UpdatedOn)
}

func (p *WorkflowRepo) GetByBoard(ctx context.Context, boardID int) (*models.Workflow, error) {
	r, err := p.queries.GetWorkflowByBoard(ctx, boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWorkflowNotFound
		}
		return nil, err
	}

	return workflowFromParts(r.ID, r.BoardID, r.BoardName, r.Name, r.Enabled, r.DryRun, r.Rules, r.CreatedOn, r.UpdatedOn)
}

func (p *WorkflowRepo) ExistsForBoard(ctx context.Context, boardID int) (bool, error) {
	return p.queries.WorkflowExistsForBoard(ctx, boardID)
}

func (p *WorkflowRepo) Insert(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	rules, err := models.EncodeWorkflowDocument(w)
	if err != nil {
		return nil, err
	}

	d, err := p.queries.InsertWorkflow(ctx, db.InsertWorkflowParams{
		BoardID: w.BoardID,
		Name:    w.Name,
		Enabled: w.Enabled,
		DryRun:  w.DryRun,
		Rules:   rules,
	})
	if err != nil {
		return nil, err
	}

	return workflowFromParts(d.ID, d.BoardID, w.BoardName, d.Name, d.Enabled, d.DryRun, d.Rules, d.CreatedOn, d.UpdatedOn)
}

func (p *WorkflowRepo) Update(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	rules, err := models.EncodeWorkflowDocument(w)
	if err != nil {
		return nil, err
	}

	d, err := p.queries.UpdateWorkflow(ctx, db.UpdateWorkflowParams{
		ID:      w.ID,
		Name:    w.Name,
		Enabled: w.Enabled,
		DryRun:  w.DryRun,
		Rules:   rules,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWorkflowNotFound
		}
		return nil, err
	}

	return workflowFromParts(d.ID, d.BoardID, w.BoardName, d.Name, d.Enabled, d.DryRun, d.Rules, d.CreatedOn, d.UpdatedOn)
}

func (p *WorkflowRepo) Delete(ctx context.Context, id int) error {
	return p.queries.DeleteWorkflow(ctx, id)
}

func workflowFromParts(id, boardID int, boardName, name string, enabled, dryRun bool, doc []byte, created, updated time.Time) (*models.Workflow, error) {
	w := &models.Workflow{
		ID:        id,
		BoardID:   boardID,
		BoardName: boardName,
		Name:      name,
		Enabled:   enabled,
		DryRun:    dryRun,
		CreatedOn: created,
		UpdatedOn: updated,
	}

	if err := models.DecodeWorkflowDocument(doc, w); err != nil {
		return nil, fmt.Errorf("decoding document for workflow %d: %w", id, err)
	}

	return w, nil
}
