package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

type appConfig struct {
	Debug              bool `json:"debug"`
	AttemptNotify      bool `json:"attempt_notify"`
	MaxMessageLength   int  `json:"max_message_length"`
	MaxConcurrentSyncs int  `json:"max_concurrent_syncs"`
}

func (cl *Client) handleGetConfig(c *gin.Context) {
	ac, err := cl.getFullConfig(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, ac)
}

func (cl *Client) handlePutConfig(c *gin.Context) {
	r := &appConfig{}
	if err := c.ShouldBindJSON(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json request"})
		return
	}

	if err := cl.updateConfigInDB(c.Request.Context(), r); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, cl.Config)
}

func (cl *Client) getFullConfig(ctx context.Context) (*appConfig, error) {
	dc, err := cl.Queries.GetAppConfig(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("no app config found, creating default")
			dc, err = cl.Queries.UpsertAppConfig(ctx, db.UpsertAppConfigParams{})
			if err != nil {
				return nil, fmt.Errorf("creating default app config: %w", err)
			}
			return dbConfigToAppConfig(dc), nil
		}
		return nil, fmt.Errorf("getting app config from db: %w", err)
	}

	return dbConfigToAppConfig(dc), nil
}

func (cl *Client) updateConfigInDB(ctx context.Context, ac *appConfig) error {
	p := configToParams(ac)

	dc, err := cl.Queries.UpsertAppConfig(ctx, p)
	if err != nil {
		return fmt.Errorf("updating in db: %w", err)
	}

	cl.Config = dbConfigToAppConfig(dc)
	setLogLevel(cl.Config.Debug)
	return nil
}

func configToParams(ac *appConfig) db.UpsertAppConfigParams {
	return db.UpsertAppConfigParams{
		Debug:              ac.Debug,
		AttemptNotify:      ac.AttemptNotify,
		MaxMessageLength:   ac.MaxMessageLength,
		MaxConcurrentSyncs: ac.MaxConcurrentSyncs,
	}
}

func dbConfigToAppConfig(dc db.AppConfig) *appConfig {
	return &appConfig{
		Debug:              dc.Debug,
		AttemptNotify:      dc.AttemptNotify,
		MaxMessageLength:   dc.MaxMessageLength,
		MaxConcurrentSyncs: dc.MaxConcurrentSyncs,
	}
}
