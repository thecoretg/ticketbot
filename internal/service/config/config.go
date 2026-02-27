package config

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

func (s *Service) ensureConfig(ctx context.Context) (*models.Config, error) {
	c, err := s.Config.Get(ctx)
	if err == nil {
		s.applyChanges(c)
		return s.ConfigRef, nil
	}

	if !errors.Is(err, models.ErrConfigNotFound) {
		return nil, fmt.Errorf("getting config from store: %w", err)
	}

	// if error was no config found, create the default
	c, err = s.Config.InsertDefault(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating default config: %w", err)
	}

	s.applyChanges(c)
	return s.ConfigRef, nil
}
