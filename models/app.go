package models

import "errors"

var ErrConfigNotFound = errors.New("config not found")

type Config struct {
	// The app will only ever have one config in the config table, so this will always just be 1.
	ID int `json:"id"`

	// MasterDryRun forces every workflow to run as a dry run: tickets are still ingested and events
	// recorded, but nothing is written to ConnectWise and no Webex messages are sent.
	MasterDryRun bool `json:"master_dry_run"`

	// CWAPIMemberIdentifier is the identifier of the ConnectWise member the API credentials belong to.
	// Updates authored by this member are ticketbot's own writes and never trigger workflows.
	// It is learned automatically from the first note ticketbot posts, but can be set by hand.
	CWAPIMemberIdentifier string `json:"cw_api_member_identifier"`

	// MaxMessageLength is the max amount of characters in a notification's ticket note output before
	// it truncates and adds a "..." to the end.
	MaxMessageLength int `json:"max_message_length"`

	// MaxConcurrentSyncs is the maximum amount of syncs that can be run at once. This defaults to 5
	// as it has been deemed a good amount to prevent constant rate limits from Connectwise.
	MaxConcurrentSyncs int `json:"max_concurrent_syncs"`

	// RequireTOTP enforces that all users must have TOTP enabled to access the application.
	RequireTOTP bool `json:"require_totp"`

	// DebugLogging enables debug-level log output at runtime without a server restart.
	DebugLogging bool `json:"debug_logging"`

	// LogRetentionDays is how many days of logs to keep in the database.
	LogRetentionDays int `json:"log_retention_days"`

	// LogCleanupIntervalHours is how often the cleanup goroutine runs to delete old logs.
	LogCleanupIntervalHours int `json:"log_cleanup_interval_hours"`

	// LogBufferSize is how many log entries to keep in the in-memory ring buffer.
	LogBufferSize int `json:"log_buffer_size"`

	// SSOEnabled turns on Microsoft Entra sign-in. It can only be enabled when the ENTRA_*
	// environment variables are set.
	SSOEnabled bool `json:"sso_enabled"`

	// PasswordLoginEnabled allows email and password sign-in. It can only be disabled while
	// SSOEnabled is true; the INITIAL_ADMIN_EMAIL account can always sign in with a password.
	PasswordLoginEnabled bool `json:"password_login_enabled"`

	// NotePreviewLength is how many characters of a new note are kept in the ticket history
	// preview. It is applied when the event is recorded, so it only affects new events.
	NotePreviewLength int `json:"note_preview_length"`
}

// ConfigUpdateParams is used for partial updates to Config. Pointer fields allow
// distinguishing between "not provided" and an explicit zero/false value.
type ConfigUpdateParams struct {
	MasterDryRun            *bool   `json:"master_dry_run"`
	CWAPIMemberIdentifier   *string `json:"cw_api_member_identifier"`
	MaxMessageLength        *int    `json:"max_message_length"`
	MaxConcurrentSyncs      *int    `json:"max_concurrent_syncs"`
	RequireTOTP             *bool   `json:"require_totp"`
	DebugLogging            *bool   `json:"debug_logging"`
	LogRetentionDays        *int    `json:"log_retention_days"`
	LogCleanupIntervalHours *int    `json:"log_cleanup_interval_hours"`
	LogBufferSize           *int    `json:"log_buffer_size"`
	SSOEnabled              *bool   `json:"sso_enabled"`
	PasswordLoginEnabled    *bool   `json:"password_login_enabled"`
	NotePreviewLength       *int    `json:"note_preview_length"`
}

var DefaultConfig = Config{
	ID:                      1,
	MasterDryRun:            true,
	MaxMessageLength:        300,
	MaxConcurrentSyncs:      5,
	RequireTOTP:             false,
	DebugLogging:            false,
	LogRetentionDays:        7,
	LogCleanupIntervalHours: 24,
	LogBufferSize:           500,
	SSOEnabled:              false,
	PasswordLoginEnabled:    true,
	NotePreviewLength:       200,
}
