package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type WebhookIntakeRepo struct {
	queries *db.Queries
}

func NewWebhookIntakeRepo(pool *pgxpool.Pool) *WebhookIntakeRepo {
	return &WebhookIntakeRepo{queries: db.New(pool)}
}

func (p *WebhookIntakeRepo) Insert(ctx context.Context, ticketID int, action models.IntakeAction, payload []byte) (*models.WebhookIntake, error) {
	row, err := p.queries.InsertWebhookIntake(ctx, db.InsertWebhookIntakeParams{
		TicketID: ticketID,
		Action:   string(action),
		Payload:  payloadBytes(payload),
	})
	if err != nil {
		return nil, err
	}
	return intakeFromPG(row), nil
}

func (p *WebhookIntakeRepo) Claim(ctx context.Context) (*models.WebhookIntake, error) {
	row, err := p.queries.ClaimWebhookIntake(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrIntakeEmpty
		}
		return nil, err
	}
	return intakeFromPG(row), nil
}

func (p *WebhookIntakeRepo) Finish(ctx context.Context, id int64) error {
	return p.queries.FinishWebhookIntake(ctx, id)
}

func (p *WebhookIntakeRepo) Reschedule(ctx context.Context, id int64, at time.Time, lastError string) error {
	return p.queries.RescheduleWebhookIntake(ctx, db.RescheduleWebhookIntakeParams{ID: id, NextAttemptAt: at, LastError: &lastError})
}

func (p *WebhookIntakeRepo) Fail(ctx context.Context, id int64, lastError string) error {
	return p.queries.FailWebhookIntake(ctx, db.FailWebhookIntakeParams{ID: id, LastError: &lastError})
}

func (p *WebhookIntakeRepo) Retry(ctx context.Context, id int64) (*models.WebhookIntake, error) {
	row, err := p.queries.RetryWebhookIntake(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrIntakeNotFound
		}
		return nil, err
	}
	return intakeFromPG(row), nil
}

func (p *WebhookIntakeRepo) Discard(ctx context.Context, id int64) (*models.WebhookIntake, error) {
	row, err := p.queries.DiscardWebhookIntake(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrIntakeNotFound
		}
		return nil, err
	}
	return intakeFromPG(row), nil
}

func (p *WebhookIntakeRepo) ResetProcessing(ctx context.Context) (int64, error) {
	return p.queries.ResetProcessingWebhookIntake(ctx)
}

func (p *WebhookIntakeRepo) List(ctx context.Context, status *models.IntakeStatus, limit int) ([]*models.WebhookIntake, error) {
	if limit <= 0 {
		limit = 100
	}
	var st *string
	if status != nil {
		s := string(*status)
		st = &s
	}
	rows, err := p.queries.ListWebhookIntake(ctx, db.ListWebhookIntakeParams{Status: st, Lim: int32(limit)})
	if err != nil {
		return nil, err
	}
	out := make([]*models.WebhookIntake, 0, len(rows))
	for _, r := range rows {
		out = append(out, intakeFromPG(r))
	}
	return out, nil
}

func (p *WebhookIntakeRepo) Stats(ctx context.Context) (*models.IntakeStats, error) {
	counts, err := p.queries.CountWebhookIntakeByStatus(ctx)
	if err != nil {
		return nil, err
	}
	st := &models.IntakeStats{Counts: make(map[models.IntakeStatus]int64, len(counts))}
	for _, c := range counts {
		st.Counts[models.IntakeStatus(c.Status)] = c.N
	}
	last, err := p.queries.LastWebhookIntakeReceivedAt(ctx)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	if err == nil {
		st.LastReceivedAt = &last
	}
	return st, nil
}

func (p *WebhookIntakeRepo) CountReceivedSince(ctx context.Context, since time.Time) (int64, error) {
	return p.queries.CountWebhookIntakeReceivedSince(ctx, since)
}

func (p *WebhookIntakeRepo) CountsByHour(ctx context.Context, since time.Time) ([]models.IntakeHourCount, error) {
	rows, err := p.queries.CountWebhookIntakeByHour(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]models.IntakeHourCount, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.IntakeHourCount{Hour: r.Hour, Count: r.N})
	}
	return out, nil
}

func (p *WebhookIntakeRepo) DeleteFinishedBefore(ctx context.Context, before time.Time) (int64, error) {
	return p.queries.DeleteFinishedWebhookIntakeBefore(ctx, &before)
}

func intakeFromPG(r *db.WebhookIntake) *models.WebhookIntake {
	return &models.WebhookIntake{
		ID:            r.ID,
		TicketID:      r.TicketID,
		Action:        models.IntakeAction(r.Action),
		Payload:       json.RawMessage(r.Payload),
		Status:        models.IntakeStatus(r.Status),
		Attempts:      r.Attempts,
		NextAttemptAt: r.NextAttemptAt,
		LastError:     r.LastError,
		ReceivedAt:    r.ReceivedAt,
		FinishedAt:    r.FinishedAt,
	}
}
