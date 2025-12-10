package config

import (
	"context"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Config    models.ConfigRepository
	ConfigRef *models.Config
}

func New(c models.ConfigRepository, cfg *models.Config) *Service {
	return &Service{
		Config:    c,
		ConfigRef: cfg,
	}
}

func (s *Service) Get(ctx context.Context) (*models.Config, error) {
	return s.ensureConfig(ctx)
}

func (s *Service) Update(ctx context.Context, p *models.Config) (*models.Config, error) {
	updated, err := s.Config.Upsert(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("upserting config in store: %w", err)
	}

	s.applyChanges(updated)
	return s.ConfigRef, nil
}

func (s *Service) applyChanges(src *models.Config) {
	cfg := s.ConfigRef
	cfg.AttemptNotify = src.AttemptNotify
	cfg.MaxConcurrentSyncs = src.MaxConcurrentSyncs
	cfg.MaxMessageLength = src.MaxMessageLength
}
