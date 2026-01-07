package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type AddigyAlertRepo struct {
	queries *db.Queries
}

func NewAddigyAlertRepo(pool *pgxpool.Pool) *AddigyAlertRepo {
	return &AddigyAlertRepo{
		queries: db.New(pool),
	}
}

func (p *AddigyAlertRepo) WithTx(tx pgx.Tx) models.AddigyAlertRepository {
	return &AddigyAlertRepo{
		queries: db.New(tx),
	}
}

func (p *AddigyAlertRepo) List(ctx context.Context) ([]*models.AddigyAlert, error) {
	dbs, err := p.queries.ListAddigyAlerts(ctx)
	if err != nil {
		return nil, err
	}

	var alerts []*models.AddigyAlert
	for _, d := range dbs {
		alerts = append(alerts, addigyAlertFromPG(d))
	}

	return alerts, nil
}

func (p *AddigyAlertRepo) ListByStatus(ctx context.Context, status string) ([]*models.AddigyAlert, error) {
	dbs, err := p.queries.ListAddigyAlertsByStatus(ctx, status)
	if err != nil {
		return nil, err
	}

	var alerts []*models.AddigyAlert
	for _, d := range dbs {
		alerts = append(alerts, addigyAlertFromPG(d))
	}

	return alerts, nil
}

func (p *AddigyAlertRepo) ListUnresolved(ctx context.Context) ([]*models.AddigyAlert, error) {
	dbs, err := p.queries.ListUnresolvedAddigyAlerts(ctx)
	if err != nil {
		return nil, err
	}

	var alerts []*models.AddigyAlert
	for _, d := range dbs {
		alerts = append(alerts, addigyAlertFromPG(d))
	}

	return alerts, nil
}

func (p *AddigyAlertRepo) ListByTicket(ctx context.Context, ticketID int) ([]*models.AddigyAlert, error) {
	dbs, err := p.queries.ListAddigyAlertsByTicket(ctx, &ticketID)
	if err != nil {
		return nil, err
	}

	var alerts []*models.AddigyAlert
	for _, d := range dbs {
		alerts = append(alerts, addigyAlertFromPG(d))
	}

	return alerts, nil
}

func (p *AddigyAlertRepo) Get(ctx context.Context, id string) (*models.AddigyAlert, error) {
	d, err := p.queries.GetAddigyAlert(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAddigyAlertNotFound
		}
		return nil, err
	}

	return addigyAlertFromPG(d), nil
}

func (p *AddigyAlertRepo) Create(ctx context.Context, a *models.AddigyAlert) (*models.AddigyAlert, error) {
	d, err := p.queries.CreateAddigyAlert(ctx, addigyAlertToCreateParams(a))
	if err != nil {
		return nil, err
	}

	return addigyAlertFromPG(d), nil
}

func (p *AddigyAlertRepo) Update(ctx context.Context, a *models.AddigyAlert) (*models.AddigyAlert, error) {
	d, err := p.queries.UpdateAddigyAlert(ctx, addigyAlertToUpdateParams(a))
	if err != nil {
		return nil, err
	}

	return addigyAlertFromPG(d), nil
}

func (p *AddigyAlertRepo) UpdateTicket(ctx context.Context, id string, ticketID *int) error {
	if err := p.queries.UpdateAddigyAlertTicket(ctx, db.UpdateAddigyAlertTicketParams{
		ID:       id,
		TicketID: ticketID,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertNotFound
		}
		return err
	}

	return nil
}

func (p *AddigyAlertRepo) UpdateStatus(ctx context.Context, id string, status string) error {
	if err := p.queries.UpdateAddigyAlertStatus(ctx, db.UpdateAddigyAlertStatusParams{
		ID:     id,
		Status: status,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertNotFound
		}
		return err
	}

	return nil
}

func (p *AddigyAlertRepo) Acknowledge(ctx context.Context, id string, acknowledgedOn time.Time) error {
	if err := p.queries.AcknowledgeAddigyAlert(ctx, db.AcknowledgeAddigyAlertParams{
		ID:             id,
		AcknowledgedOn: &acknowledgedOn,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertNotFound
		}
		return err
	}

	return nil
}

func (p *AddigyAlertRepo) Resolve(ctx context.Context, id string, resolvedOn time.Time, resolvedByEmail string) error {
	if err := p.queries.ResolveAddigyAlert(ctx, db.ResolveAddigyAlertParams{
		ID:              id,
		ResolvedOn:      &resolvedOn,
		ResolvedByEmail: &resolvedByEmail,
	}); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertNotFound
		}
		return err
	}

	return nil
}

func (p *AddigyAlertRepo) Delete(ctx context.Context, id string) error {
	if err := p.queries.DeleteAddigyAlert(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertNotFound
		}
		return err
	}

	return nil
}

func addigyAlertToCreateParams(a *models.AddigyAlert) db.CreateAddigyAlertParams {
	return db.CreateAddigyAlertParams{
		ID:              a.ID,
		TicketID:        a.TicketID,
		Level:           a.Level,
		Category:        a.Category,
		Name:            a.Name,
		FactName:        a.FactName,
		FactIdentifier:  a.FactIdentifier,
		FactType:        a.FactType,
		Selector:        a.Selector,
		Status:          a.Status,
		Value:           a.Value,
		Muted:           a.Muted,
		Remediation:     a.Remediation,
		ResolvedByEmail: a.ResolvedByEmail,
		ResolvedOn:      a.ResolvedOn,
		AcknowledgedOn:  a.AcknowledgedOn,
		AddedOn:         a.AddedOn,
	}
}

func addigyAlertToUpdateParams(a *models.AddigyAlert) db.UpdateAddigyAlertParams {
	return db.UpdateAddigyAlertParams{
		ID:              a.ID,
		TicketID:        a.TicketID,
		Level:           a.Level,
		Category:        a.Category,
		Name:            a.Name,
		FactName:        a.FactName,
		FactIdentifier:  a.FactIdentifier,
		FactType:        a.FactType,
		Selector:        a.Selector,
		Status:          a.Status,
		Value:           a.Value,
		Muted:           a.Muted,
		Remediation:     a.Remediation,
		ResolvedByEmail: a.ResolvedByEmail,
		ResolvedOn:      a.ResolvedOn,
		AcknowledgedOn:  a.AcknowledgedOn,
	}
}

func addigyAlertFromPG(pg *db.AddigyAlert) *models.AddigyAlert {
	return &models.AddigyAlert{
		ID:              pg.ID,
		TicketID:        pg.TicketID,
		Level:           pg.Level,
		Category:        pg.Category,
		Name:            pg.Name,
		FactName:        pg.FactName,
		FactIdentifier:  pg.FactIdentifier,
		FactType:        pg.FactType,
		Selector:        pg.Selector,
		Status:          pg.Status,
		Value:           pg.Value,
		Muted:           pg.Muted,
		Remediation:     pg.Remediation,
		ResolvedByEmail: pg.ResolvedByEmail,
		ResolvedOn:      pg.ResolvedOn,
		AcknowledgedOn:  pg.AcknowledgedOn,
		AddedOn:         pg.AddedOn,
	}
}
