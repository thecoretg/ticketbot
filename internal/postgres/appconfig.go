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
		AttemptNotify:      c.AttemptNotify,
		MaxMessageLength:   c.MaxMessageLength,
		MaxConcurrentSyncs: c.MaxConcurrentSyncs,
		SkipLaunchSyncs:    c.SkipLaunchSyncs,
	}
}

func configFromPG(pg *db.AppConfig) *models.Config {
	return &models.Config{
		ID:                 pg.ID,
		AttemptNotify:      pg.AttemptNotify,
		MaxMessageLength:   pg.MaxMessageLength,
		MaxConcurrentSyncs: pg.MaxConcurrentSyncs,
		SkipLaunchSyncs:    pg.SkipLaunchSyncs,
	}
}
