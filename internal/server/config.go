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

// appConfigPayload is used for partial updates to the app config
type appConfigPayload struct {
	Debug              *bool `json:"debug,omitempty"`
	AttemptNotify      *bool `json:"attempt_notify,omitempty"`
	MaxMessageLength   *int  `json:"max_message_length,omitempty"`
	MaxConcurrentSyncs *int  `json:"max_concurrent_syncs,omitempty"`
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
	r := &appConfigPayload{}
	if err := c.ShouldBindJSON(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json request"})
		return
	}

	mc := mergeConfigPayload(cl.Config, r)
	cl.Config = mc

	if err := cl.updateConfigInDB(c.Request.Context()); err != nil {
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
			dc, err = cl.Queries.InsertDefaultAppConfig(ctx)
			if err != nil {
				return nil, fmt.Errorf("creating default app config: %w", err)
			}
			return dbConfigToAppConfig(dc), nil
		}
		return nil, fmt.Errorf("getting app config from db: %w", err)
	}

	return dbConfigToAppConfig(dc), nil
}

func mergeConfigPayload(ac *appConfig, p *appConfigPayload) *appConfig {
	merged := *ac
	if p.Debug != nil {
		merged.Debug = *p.Debug
	}

	if p.AttemptNotify != nil {
		merged.AttemptNotify = *p.AttemptNotify
	}

	if p.MaxMessageLength != nil {
		merged.MaxMessageLength = *p.MaxMessageLength
	}

	if p.MaxConcurrentSyncs != nil {
		merged.MaxConcurrentSyncs = *p.MaxConcurrentSyncs
	}

	return &merged
}

func (cl *Client) updateConfigInDB(ctx context.Context) error {
	p := configToParams(cl.Config)
	if _, err := cl.Queries.UpsertAppConfig(ctx, p); err != nil {
		return fmt.Errorf("updating in db: %w", err)
	}

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
