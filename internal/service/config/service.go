package config

import (
	"context"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/repos"
)

type Service struct {
	Config    repos.ConfigRepository
	ConfigRef *models.Config
}

func New(c repos.ConfigRepository, cfg *models.Config) *Service {
	return &Service{
		Config:    c,
		ConfigRef: cfg,
	}
}

func (s *Service) Get(ctx context.Context) (*models.Config, error) {
	return s.ensureConfig(ctx)
}

func (s *Service) Update(ctx context.Context, p *models.ConfigUpdateParams) (*models.Config, error) {
	current, err := s.ensureConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting current config: %w", err)
	}

	merged := *current
	if p.AttemptNotify != nil {
		merged.AttemptNotify = *p.AttemptNotify
	}
	if p.MaxMessageLength != nil {
		merged.MaxMessageLength = *p.MaxMessageLength
	}
	if p.MaxConcurrentSyncs != nil {
		merged.MaxConcurrentSyncs = *p.MaxConcurrentSyncs
	}
	if p.SkipLaunchSyncs != nil {
		merged.SkipLaunchSyncs = *p.SkipLaunchSyncs
	}
	if p.RequireTOTP != nil {
		merged.RequireTOTP = *p.RequireTOTP
	}

	updated, err := s.Config.Upsert(ctx, &merged)
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
	cfg.RequireTOTP = src.RequireTOTP
}
