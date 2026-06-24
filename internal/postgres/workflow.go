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

type WorkflowRepo struct {
	queries *db.Queries
}

func NewWorkflowRepo(pool *pgxpool.Pool) *WorkflowRepo {
	return &WorkflowRepo{
		queries: db.New(pool),
	}
}

func (p *WorkflowRepo) WithTx(tx pgx.Tx) repos.WorkflowRepository {
	return &WorkflowRepo{
		queries: db.New(tx),
	}
}

func (p *WorkflowRepo) ListAll(ctx context.Context) ([]*models.Workflow, error) {
	dm, err := p.queries.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	b := make([]*models.Workflow, 0, len(dm))
	for _, d := range dm {
		b = append(b, workflowFromPG(d))
	}

	return b, nil
}

func (p *WorkflowRepo) ListEnabled(ctx context.Context) ([]*models.Workflow, error) {
	dm, err := p.queries.ListEnabledWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	b := make([]*models.Workflow, 0, len(dm))
	for _, d := range dm {
		b = append(b, workflowFromPG(d))
	}

	return b, nil
}

func (p *WorkflowRepo) Get(ctx context.Context, id int) (*models.Workflow, error) {
	d, err := p.queries.GetWorkflow(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWorkflowNotFound
		}
		return nil, err
	}

	return workflowFromPG(d), nil
}

func (p *WorkflowRepo) Exists(ctx context.Context, id int) (bool, error) {
	return p.queries.CheckWorkflowExists(ctx, id)
}

func (p *WorkflowRepo) Insert(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	d, err := p.queries.InsertWorkflow(ctx, db.InsertWorkflowParams{
		Name:           w.Name,
		CwBoardID:      w.CwBoardID,
		OnTicketAction: w.OnTicketAction,
		Conditions:     conditionsBytes(w.Root),
		Actions:        actionsBytes(w.Actions),
		Priority:       w.Priority,
		Enabled:        w.Enabled,
		SimulationMode: w.SimulationMode,
	})
	if err != nil {
		return nil, err
	}

	return workflowFromPG(d), nil
}

func (p *WorkflowRepo) Update(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	d, err := p.queries.UpdateWorkflow(ctx, db.UpdateWorkflowParams{
		ID:             w.ID,
		Name:           w.Name,
		CwBoardID:      w.CwBoardID,
		OnTicketAction: w.OnTicketAction,
		Conditions:     conditionsBytes(w.Root),
		Actions:        actionsBytes(w.Actions),
		Priority:       w.Priority,
		Enabled:        w.Enabled,
		SimulationMode: w.SimulationMode,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrWorkflowNotFound
		}
		return nil, err
	}

	return workflowFromPG(d), nil
}

func (p *WorkflowRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteWorkflow(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrWorkflowNotFound
		}
		return err
	}

	return nil
}

func workflowFromPG(pg *db.Workflow) *models.Workflow {
	return &models.Workflow{
		ID:             pg.ID,
		Name:           pg.Name,
		CwBoardID:      pg.CwBoardID,
		OnTicketAction: pg.OnTicketAction,
		Root:           conditionsFromBytes(pg.Conditions),
		Actions:        actionsFromBytes(pg.Actions),
		Priority:       pg.Priority,
		Enabled:        pg.Enabled,
		SimulationMode: pg.SimulationMode,
		CreatedOn:      pg.CreatedOn,
		UpdatedOn:      pg.UpdatedOn,
	}
}

// actionsBytes marshals a workflow's ordered action list to the NOT NULL jsonb
// column, storing an empty array when there are none.
func actionsBytes(actions []models.Action) []byte {
	if len(actions) == 0 {
		return []byte("[]")
	}
	b, err := json.Marshal(actions)
	if err != nil {
		return []byte("[]")
	}
	return b
}

func actionsFromBytes(b []byte) []models.Action {
	if len(b) == 0 {
		return nil
	}
	var actions []models.Action
	if err := json.Unmarshal(b, &actions); err != nil {
		return nil
	}
	return actions
}

// conditionsBytes marshals the optional root condition group to the NOT NULL
// jsonb column, storing an empty object when there is no condition tree.
func conditionsBytes(root *models.ConditionGroup) []byte {
	if root == nil {
		return []byte("{}")
	}
	b, err := json.Marshal(root)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// conditionsFromBytes unmarshals the stored root condition group, treating an
// empty object or empty bytes as "no conditions" (nil root).
func conditionsFromBytes(b []byte) *models.ConditionGroup {
	if len(b) == 0 {
		return nil
	}
	var root models.ConditionGroup
	if err := json.Unmarshal(b, &root); err != nil {
		return nil
	}
	if root.Operator == "" && len(root.Children) == 0 {
		return nil
	}
	return &root
}

type WorkflowRunRepo struct {
	queries *db.Queries
}

func NewWorkflowRunRepo(pool *pgxpool.Pool) *WorkflowRunRepo {
	return &WorkflowRunRepo{
		queries: db.New(pool),
	}
}

func (p *WorkflowRunRepo) WithTx(tx pgx.Tx) repos.WorkflowRunRepository {
	return &WorkflowRunRepo{
		queries: db.New(tx),
	}
}

func (p *WorkflowRunRepo) Exists(ctx context.Context, ticketID, workflowID, actionIndex int) (bool, error) {
	return p.queries.CheckWorkflowRunExists(ctx, db.CheckWorkflowRunExistsParams{
		TicketID:    ticketID,
		WorkflowID:  workflowID,
		ActionIndex: actionIndex,
	})
}

func (p *WorkflowRunRepo) Insert(ctx context.Context, ticketID, workflowID, actionIndex int) error {
	return p.queries.InsertWorkflowRun(ctx, db.InsertWorkflowRunParams{
		TicketID:    ticketID,
		WorkflowID:  workflowID,
		ActionIndex: actionIndex,
	})
}
