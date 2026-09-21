package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// getStartupConfig returns the stored config, inserting the default only when none exists yet.
func getStartupConfig(ctx context.Context, r repos.ConfigRepository) (*models.Config, error) {
	cfg, err := r.Get(ctx)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, models.ErrConfigNotFound) {
		return nil, fmt.Errorf("getting config: %w", err)
	}

	slog.Info("no config found in store; inserting default")
	cfg, err = r.Upsert(ctx, &models.DefaultConfig)
	if err != nil {
		return nil, fmt.Errorf("inserting default config: %w", err)
	}
	return cfg, nil
}

// makeMessageSender is always the real Webex client. Keeping messages away from real rooms is the
// redirect room's job (app config), not an environment switch.
func makeMessageSender(ctx context.Context, webexSecret string) (repos.MessageSender, error) {
	return webex.NewClient(ctx, webex.Config{Token: webexSecret})
}
