package config

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Config    repos.ConfigRepository
	ConfigRef *models.Config
	level     *slog.LevelVar
	logBuf    *logging.BufferHandler
}

func New(c repos.ConfigRepository, cfg *models.Config, level *slog.LevelVar, logBuf *logging.BufferHandler) *Service {
	s := &Service{
		Config:    c,
		ConfigRef: cfg,
		level:     level,
		logBuf:    logBuf,
	}
	s.applyChanges(cfg)
	return s
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
	if p.RequireTOTP != nil {
		merged.RequireTOTP = *p.RequireTOTP
	}
	if p.DebugLogging != nil {
		merged.DebugLogging = *p.DebugLogging
	}
	if p.LogRetentionDays != nil {
		merged.LogRetentionDays = *p.LogRetentionDays
	}
	if p.LogCleanupIntervalHours != nil {
		merged.LogCleanupIntervalHours = *p.LogCleanupIntervalHours
	}
	if p.LogBufferSize != nil {
		merged.LogBufferSize = *p.LogBufferSize
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
	cfg.DebugLogging = src.DebugLogging
	cfg.LogRetentionDays = src.LogRetentionDays
	cfg.LogCleanupIntervalHours = src.LogCleanupIntervalHours
	cfg.LogBufferSize = src.LogBufferSize

	if s.logBuf != nil && src.LogBufferSize > 0 && src.LogBufferSize != s.logBuf.Size() {
		s.logBuf.Resize(src.LogBufferSize)
		slog.Info("log buffer resized", "size", src.LogBufferSize)
	}

	if s.level != nil {
		if src.DebugLogging {
			s.level.Set(slog.LevelDebug)
		} else {
			s.level.Set(slog.LevelInfo)
		}
	}
}
