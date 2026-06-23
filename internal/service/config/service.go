package config

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// connTestTimeout bounds each credential check so a hung endpoint can't block the request.
const connTestTimeout = 10 * time.Second

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
	if p.AttemptWorkflow != nil {
		merged.AttemptWorkflow = *p.AttemptWorkflow
	}
	if p.CwBotMemberIdentifier != nil {
		merged.CwBotMemberIdentifier = *p.CwBotMemberIdentifier
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
	if p.RootURL != nil {
		merged.RootURL = *p.RootURL
	}
	if p.CwCompanyID != nil {
		merged.CwCompanyID = *p.CwCompanyID
	}
	if p.CwClientID != nil {
		merged.CwClientID = *p.CwClientID
	}
	if p.CwPublicKey != nil {
		merged.CwPublicKey = *p.CwPublicKey
	}
	// Secrets are write-only: an empty/omitted value means "leave unchanged" so the
	// panel never has to round-trip the secret back to the browser to preserve it.
	if p.CwPrivateKey != nil && *p.CwPrivateKey != "" {
		merged.CwPrivateKey = *p.CwPrivateKey
	}
	if p.WebexSecret != nil && *p.WebexSecret != "" {
		merged.WebexSecret = *p.WebexSecret
	}

	updated, err := s.Config.Upsert(ctx, &merged)
	if err != nil {
		return nil, fmt.Errorf("upserting config in store: %w", err)
	}

	s.applyChanges(updated)
	return s.ConfigRef, nil
}

// ServiceCheck is the result of a single credential connection test.
type ServiceCheck struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// ConnTestResult reports whether the stored Connectwise and Webex credentials can
// authenticate against their APIs.
type ConnTestResult struct {
	CW    ServiceCheck `json:"cw"`
	Webex ServiceCheck `json:"webex"`
}

// TestConnections builds ephemeral clients from the currently-stored credentials
// and performs a cheap authenticated GET against each. It tests the saved config
// (not the running clients), so freshly-saved credentials can be verified before a
// restart applies them.
func (s *Service) TestConnections(ctx context.Context) (ConnTestResult, error) {
	cfg, err := s.ensureConfig(ctx)
	if err != nil {
		return ConnTestResult{}, fmt.Errorf("getting current config: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, connTestTimeout)
	defer cancel()

	return ConnTestResult{
		CW:    testCW(ctx, cfg),
		Webex: testWebex(ctx, cfg),
	}, nil
}

func testCW(ctx context.Context, cfg *models.Config) ServiceCheck {
	client, err := psa.NewClient(ctx, psa.Config{
		PublicKey:  cfg.CwPublicKey,
		PrivateKey: cfg.CwPrivateKey,
		ClientID:   cfg.CwClientID,
		CompanyID:  cfg.CwCompanyID,
	})
	if err != nil {
		return ServiceCheck{Error: err.Error()}
	}
	if _, err := client.ListBoards(ctx, map[string]string{"pageSize": "1"}); err != nil {
		return ServiceCheck{Error: err.Error()}
	}
	return ServiceCheck{OK: true}
}

func testWebex(ctx context.Context, cfg *models.Config) ServiceCheck {
	client, err := webex.NewClient(ctx, webex.Config{Token: cfg.WebexSecret})
	if err != nil {
		return ServiceCheck{Error: err.Error()}
	}
	if _, err := client.ListRooms(ctx, map[string]string{"max": "1"}); err != nil {
		return ServiceCheck{Error: err.Error()}
	}
	return ServiceCheck{OK: true}
}

func (s *Service) applyChanges(src *models.Config) {
	cfg := s.ConfigRef
	cfg.AttemptNotify = src.AttemptNotify
	cfg.AttemptWorkflow = src.AttemptWorkflow
	cfg.CwBotMemberIdentifier = src.CwBotMemberIdentifier
	cfg.MaxConcurrentSyncs = src.MaxConcurrentSyncs
	cfg.MaxMessageLength = src.MaxMessageLength
	cfg.RequireTOTP = src.RequireTOTP
	cfg.DebugLogging = src.DebugLogging
	cfg.LogRetentionDays = src.LogRetentionDays
	cfg.LogCleanupIntervalHours = src.LogCleanupIntervalHours
	cfg.LogBufferSize = src.LogBufferSize
	cfg.RootURL = src.RootURL
	cfg.CwCompanyID = src.CwCompanyID
	cfg.CwClientID = src.CwClientID
	cfg.CwPublicKey = src.CwPublicKey
	cfg.CwPrivateKey = src.CwPrivateKey
	cfg.WebexSecret = src.WebexSecret

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
