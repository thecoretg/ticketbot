package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowRunRepo struct {
	queries *db.Queries
}

func NewWorkflowRunRepo(pool *pgxpool.Pool) *WorkflowRunRepo {
	return &WorkflowRunRepo{queries: db.New(pool)}
}

func (p *WorkflowRunRepo) Insert(ctx context.Context, r *models.WorkflowRun) error {
	return p.queries.InsertWorkflowRun(ctx, db.InsertWorkflowRunParams{
		RunID:          r.RunID,
		TicketID:       r.TicketID,
		BoardID:        r.BoardID,
		WorkflowID:     r.WorkflowID,
		WorkflowName:   r.WorkflowName,
		Event:          string(r.Event),
		Source:         string(r.Source),
		DryRun:         r.DryRun,
		StartedAt:      r.StartedAt,
		DurationMs:     r.DurationMs,
		Steps:          r.Steps,
		Actions:        r.Actions,
		Writes:         r.Writes,
		NotifSent:      r.NotifSent,
		NotifWouldSend: r.NotifWouldSend,
		NotifNone:      r.NotifNone,
		Errors:         r.Errors,
		Outcome:        string(r.Outcome),
	})
}

func (p *WorkflowRunRepo) Get(ctx context.Context, runID string) (*models.WorkflowRun, error) {
	row, err := p.queries.GetWorkflowRun(ctx, runID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrRunNotFound
		}
		return nil, err
	}
	return runFromPG(row), nil
}

func (p *WorkflowRunRepo) List(ctx context.Context, f models.RunFilter) ([]*models.WorkflowRun, error) {
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 100
	}
	var outcome *string
	if f.Outcome != "" {
		s := string(f.Outcome)
		outcome = &s
	}
	rows, err := p.queries.ListWorkflowRuns(ctx, db.ListWorkflowRunsParams{
		BoardID:  f.BoardID,
		Outcome:  outcome,
		TicketID: f.TicketID,
		FromAt:   f.From,
		ToAt:     f.To,
		BeforeAt: f.Before,
		Lim:      int32(f.Limit),
	})
	if err != nil {
		return nil, err
	}
	out := make([]*models.WorkflowRun, 0, len(rows))
	for _, r := range rows {
		out = append(out, runFromPG(r))
	}
	return out, nil
}

func (p *WorkflowRunRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return p.queries.DeleteWorkflowRunsBefore(ctx, before)
}

func runFromPG(r *db.WorkflowRun) *models.WorkflowRun {
	return &models.WorkflowRun{
		RunID:          r.RunID,
		TicketID:       r.TicketID,
		BoardID:        r.BoardID,
		WorkflowID:     r.WorkflowID,
		WorkflowName:   r.WorkflowName,
		Event:          models.TriggerEvent(r.Event),
		Source:         models.EventSource(r.Source),
		DryRun:         r.DryRun,
		StartedAt:      r.StartedAt,
		DurationMs:     r.DurationMs,
		Steps:          r.Steps,
		Actions:        r.Actions,
		Writes:         r.Writes,
		NotifSent:      r.NotifSent,
		NotifWouldSend: r.NotifWouldSend,
		NotifNone:      r.NotifNone,
		Errors:         r.Errors,
		Outcome:        models.RunOutcome(r.Outcome),
	}
}
