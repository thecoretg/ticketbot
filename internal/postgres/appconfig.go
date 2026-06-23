package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type ConfigRepo struct {
	queries *db.Queries
}

func NewConfigRepo(pool *pgxpool.Pool) *ConfigRepo {
	return &ConfigRepo{
		queries: db.New(pool),
	}
}

func (p *ConfigRepo) WithTx(tx pgx.Tx) repos.ConfigRepository {
	return &ConfigRepo{
		queries: db.New(tx),
	}
}

func (p *ConfigRepo) Get(ctx context.Context) (*models.Config, error) {
	d, err := p.queries.GetAppConfig(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrConfigNotFound
		}
		return nil, err
	}

	return configFromPG(d), nil
}

func (p *ConfigRepo) InsertDefault(ctx context.Context) (*models.Config, error) {
	d, err := p.queries.InsertDefaultAppConfig(ctx)
	if err != nil {
		return nil, err
	}

	return configFromPG(d), nil
}

func (p *ConfigRepo) Upsert(ctx context.Context, c *models.Config) (*models.Config, error) {
	d, err := p.queries.UpsertAppConfig(ctx, configToUpsertParams(c))
	if err != nil {
		return nil, err
	}

	return configFromPG(d), nil
}

func configToUpsertParams(c *models.Config) db.UpsertAppConfigParams {
	return db.UpsertAppConfigParams{
		AttemptNotify:           c.AttemptNotify,
		MaxMessageLength:        c.MaxMessageLength,
		MaxConcurrentSyncs:      c.MaxConcurrentSyncs,
		RequireTotp:             c.RequireTOTP,
		DebugLogging:            c.DebugLogging,
		LogRetentionDays:        c.LogRetentionDays,
		LogCleanupIntervalHours: c.LogCleanupIntervalHours,
		LogBufferSize:           c.LogBufferSize,
		AttemptWorkflow:         c.AttemptWorkflow,
		CwBotMemberIdentifier:   c.CwBotMemberIdentifier,
		RootUrl:                 c.RootURL,
		CwCompanyID:             c.CwCompanyID,
		CwClientID:              c.CwClientID,
		CwPublicKey:             c.CwPublicKey,
		CwPrivateKey:            c.CwPrivateKey,
		WebexSecret:             c.WebexSecret,
	}
}

func configFromPG(pg *db.AppConfig) *models.Config {
	return &models.Config{
		ID:                      pg.ID,
		AttemptNotify:           pg.AttemptNotify,
		MaxMessageLength:        pg.MaxMessageLength,
		MaxConcurrentSyncs:      pg.MaxConcurrentSyncs,
		RequireTOTP:             pg.RequireTotp,
		DebugLogging:            pg.DebugLogging,
		LogRetentionDays:        pg.LogRetentionDays,
		LogCleanupIntervalHours: pg.LogCleanupIntervalHours,
		LogBufferSize:           pg.LogBufferSize,
		AttemptWorkflow:         pg.AttemptWorkflow,
		CwBotMemberIdentifier:   pg.CwBotMemberIdentifier,
		RootURL:                 pg.RootUrl,
		CwCompanyID:             pg.CwCompanyID,
		CwClientID:              pg.CwClientID,
		CwPublicKey:             pg.CwPublicKey,
		CwPrivateKey:            pg.CwPrivateKey,
		WebexSecret:             pg.WebexSecret,
	}
}
