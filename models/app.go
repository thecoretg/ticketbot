package models

import "errors"

var ErrConfigNotFound = errors.New("config not found")

type Config struct {
	// The app will only ever have one config in the config table, so this will always just be 1.
	ID int `json:"id"`

	// AttemptNotify is a full killswitch for attempting to process notifications for tickets. If it is off,
	// all notifier rules will be disregarded.
	AttemptNotify bool `json:"attempt_notify"`

	// AttemptWorkflow is a full killswitch for the ticket workflow pipeline. If it is off, no
	// workflows run and tickets are processed exactly as before.
	AttemptWorkflow bool `json:"attempt_workflow"`

	// CwBotMemberIdentifier is the Connectwise member identifier the bot uses when it writes to tickets
	// (e.g. authoring notes). It is also used to detect and skip the bot's own webhooks, preventing
	// workflow loops.
	CwBotMemberIdentifier string `json:"cw_bot_member_identifier"`

	// The following credential/connection fields can be managed from the admin panel.
	// When the matching environment variable is set it takes precedence and is written
	// back here on startup (see server.mergeEnvCreds). RootURL is the externally
	// reachable base URL used for Connectwise webhook callbacks.
	RootURL string `json:"root_url"`

	// Connectwise PSA API credentials.
	CwCompanyID  string `json:"cw_company_id"`
	CwClientID   string `json:"cw_client_id"`
	CwPublicKey  string `json:"cw_public_key"`
	CwPrivateKey string `json:"cw_private_key"` // secret: scrubbed from GET responses

	// WebexSecret is the Webex bot bearer token.
	WebexSecret string `json:"webex_secret"` // secret: scrubbed from GET responses

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
}

// ConfigUpdateParams is used for partial updates to Config. Pointer fields allow
// distinguishing between "not provided" and an explicit zero/false value.
type ConfigUpdateParams struct {
	AttemptNotify           *bool   `json:"attempt_notify"`
	MaxMessageLength        *int    `json:"max_message_length"`
	MaxConcurrentSyncs      *int    `json:"max_concurrent_syncs"`
	RequireTOTP             *bool   `json:"require_totp"`
	DebugLogging            *bool   `json:"debug_logging"`
	LogRetentionDays        *int    `json:"log_retention_days"`
	LogCleanupIntervalHours *int    `json:"log_cleanup_interval_hours"`
	LogBufferSize           *int    `json:"log_buffer_size"`
	AttemptWorkflow         *bool   `json:"attempt_workflow"`
	CwBotMemberIdentifier   *string `json:"cw_bot_member_identifier"`
	RootURL                 *string `json:"root_url"`
	CwCompanyID             *string `json:"cw_company_id"`
	CwClientID              *string `json:"cw_client_id"`
	CwPublicKey             *string `json:"cw_public_key"`
	CwPrivateKey            *string `json:"cw_private_key"`
	WebexSecret             *string `json:"webex_secret"`
}

var DefaultConfig = Config{
	ID:                      1,
	AttemptNotify:           false,
	AttemptWorkflow:         false,
	CwBotMemberIdentifier:   "",
	MaxMessageLength:        300,
	MaxConcurrentSyncs:      5,
	RequireTOTP:             false,
	DebugLogging:            false,
	LogRetentionDays:        7,
	LogCleanupIntervalHours: 24,
	LogBufferSize:           500,
}
