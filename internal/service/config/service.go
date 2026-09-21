package config

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

var (
	ErrSSONotConfigured      = errors.New("sso cannot be enabled until ENTRA_TENANT_ID, ENTRA_CLIENT_ID and ENTRA_CLIENT_SECRET are set")
	ErrPasswordLoginNeedsSSO = errors.New("password sign-in can only be disabled while sso is enabled")
	ErrNotePreviewLength     = errors.New("note preview length must be at least 1 character")
	ErrWriteCap              = errors.New("write cap per ticket cannot be negative")
	ErrStaleMinutes          = errors.New("stale alert minutes cannot be negative")
	ErrHistoryRetention      = errors.New("history retention days cannot be negative")
)

// ValidationError marks a rejected update the caller should report as a bad request.
type ValidationError struct{ Err error }

func (e ValidationError) Error() string { return e.Err.Error() }
func (e ValidationError) Unwrap() error { return e.Err }

type Service struct {
	Config    repos.ConfigRepository
	ConfigRef *models.Config
	level     *slog.LevelVar
	logBuf    *logging.BufferHandler
	// ssoConfigured is whether the ENTRA_* environment variables are present.
	ssoConfigured bool
}

func New(c repos.ConfigRepository, cfg *models.Config, level *slog.LevelVar, logBuf *logging.BufferHandler, ssoConfigured bool) *Service {
	s := &Service{
		Config:        c,
		ConfigRef:     cfg,
		level:         level,
		logBuf:        logBuf,
		ssoConfigured: ssoConfigured,
	}
	s.applyChanges(cfg)
	return s
}

// validate rejects toggle combinations that would lock everyone out.
func (s *Service) validate(c *models.Config) error {
	if c.NotePreviewLength < 1 {
		return ValidationError{ErrNotePreviewLength}
	}
	if c.WriteCapPerTicket < 0 {
		return ValidationError{ErrWriteCap}
	}
	if c.StaleAlertMinutes < 0 {
		return ValidationError{ErrStaleMinutes}
	}
	if c.HistoryRetentionDays < 0 {
		return ValidationError{ErrHistoryRetention}
	}
	if c.SSOEnabled && !s.ssoConfigured {
		return ValidationError{ErrSSONotConfigured}
	}
	if !c.PasswordLoginEnabled && !c.SSOEnabled {
		return ValidationError{ErrPasswordLoginNeedsSSO}
	}
	return nil
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
	if p.MasterDryRun != nil {
		merged.MasterDryRun = *p.MasterDryRun
	}
	if p.CWAPIMemberIdentifier != nil {
		merged.CWAPIMemberIdentifier = *p.CWAPIMemberIdentifier
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
	if p.SSOEnabled != nil {
		merged.SSOEnabled = *p.SSOEnabled
	}
	if p.PasswordLoginEnabled != nil {
		merged.PasswordLoginEnabled = *p.PasswordLoginEnabled
	}
	if p.NotePreviewLength != nil {
		merged.NotePreviewLength = *p.NotePreviewLength
	}
	if p.WriteCapPerTicket != nil {
		merged.WriteCapPerTicket = *p.WriteCapPerTicket
	}
	if p.OpsRoomID != nil {
		merged.OpsRoomID = optionalID(*p.OpsRoomID)
	}
	if p.RedirectRoomID != nil {
		merged.RedirectRoomID = optionalID(*p.RedirectRoomID)
	}
	if p.StaleAlertMinutes != nil {
		merged.StaleAlertMinutes = *p.StaleAlertMinutes
	}
	if p.HistoryRetentionDays != nil {
		merged.HistoryRetentionDays = *p.HistoryRetentionDays
	}

	if err := s.validate(&merged); err != nil {
		return nil, err
	}

	updated, err := s.Config.Upsert(ctx, &merged)
	if err != nil {
		return nil, fmt.Errorf("upserting config in store: %w", err)
	}

	s.applyChanges(updated)
	return s.ConfigRef, nil
}

// optionalID turns the update wire form (0 clears) into the stored form (nil).
func optionalID(id int) *int {
	if id <= 0 {
		return nil
	}
	return &id
}

func (s *Service) applyChanges(src *models.Config) {
	cfg := s.ConfigRef
	cfg.MasterDryRun = src.MasterDryRun
	cfg.CWAPIMemberIdentifier = src.CWAPIMemberIdentifier
	cfg.MaxConcurrentSyncs = src.MaxConcurrentSyncs
	cfg.MaxMessageLength = src.MaxMessageLength
	cfg.RequireTOTP = src.RequireTOTP
	cfg.DebugLogging = src.DebugLogging
	cfg.LogRetentionDays = src.LogRetentionDays
	cfg.LogCleanupIntervalHours = src.LogCleanupIntervalHours
	cfg.LogBufferSize = src.LogBufferSize
	cfg.SSOEnabled = src.SSOEnabled
	cfg.PasswordLoginEnabled = src.PasswordLoginEnabled
	cfg.NotePreviewLength = src.NotePreviewLength
	cfg.WriteCapPerTicket = src.WriteCapPerTicket
	cfg.OpsRoomID = src.OpsRoomID
	cfg.RedirectRoomID = src.RedirectRoomID
	cfg.StaleAlertMinutes = src.StaleAlertMinutes
	cfg.HistoryRetentionDays = src.HistoryRetentionDays

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
