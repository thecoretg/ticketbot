package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type AddigyAlertConfigRepo struct {
	queries *db.Queries
}

func NewAddigyAlertConfigRepo(pool *pgxpool.Pool) *AddigyAlertConfigRepo {
	return &AddigyAlertConfigRepo{
		queries: db.New(pool),
	}
}

func (p *AddigyAlertConfigRepo) WithTx(tx pgx.Tx) models.AddigyAlertConfigRepository {
	return &AddigyAlertConfigRepo{
		queries: db.New(tx),
	}
}

func (p *AddigyAlertConfigRepo) Get(ctx context.Context) (*models.AddigyAlertConfig, error) {
	d, err := p.queries.GetAddigyAlertConfig(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrAddigyAlertConfigNotFound
		}
		return nil, err
	}

	return addigyAlertConfigFromPG(d), nil
}

func (p *AddigyAlertConfigRepo) Upsert(ctx context.Context, c *models.AddigyAlertConfig) (*models.AddigyAlertConfig, error) {
	d, err := p.queries.UpsertAddigyAlertConfig(ctx, addigyAlertConfigToUpsertParams(c))
	if err != nil {
		return nil, err
	}

	return addigyAlertConfigFromPG(d), nil
}

func (p *AddigyAlertConfigRepo) Delete(ctx context.Context) error {
	if err := p.queries.DeleteAddigyAlertConfig(ctx); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrAddigyAlertConfigNotFound
		}
		return err
	}

	return nil
}

func addigyAlertConfigToUpsertParams(c *models.AddigyAlertConfig) db.UpsertAddigyAlertConfigParams {
	return db.UpsertAddigyAlertConfigParams{
		CwBoardID:            c.CWBoardID,
		UnattendedStatusID:   c.UnattendedStatusID,
		AcknowledgedStatusID: c.AcknowledgedStatusID,
		Mute1DayStatusID:     c.Mute1DayStatusID,
		Mute5DayStatusID:     c.Mute5DayStatusID,
		Mute10DayStatusID:    c.Mute10DayStatusID,
		Mute30DayStatusID:    c.Mute30DayStatusID,
	}
}

func addigyAlertConfigFromPG(pg *db.AddigyAlertConfig) *models.AddigyAlertConfig {
	return &models.AddigyAlertConfig{
		ID:                   pg.ID,
		CWBoardID:            pg.CwBoardID,
		UnattendedStatusID:   pg.UnattendedStatusID,
		AcknowledgedStatusID: pg.AcknowledgedStatusID,
		Mute1DayStatusID:     pg.Mute1DayStatusID,
		Mute5DayStatusID:     pg.Mute5DayStatusID,
		Mute10DayStatusID:    pg.Mute10DayStatusID,
		Mute30DayStatusID:    pg.Mute30DayStatusID,
		UpdatedOn:            pg.UpdatedOn,
		AddedOn:              pg.AddedOn,
	}
}
