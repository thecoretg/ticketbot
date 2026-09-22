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

	// WriteCapPerTicket is how many ConnectWise writes workflows may make to one ticket in a
	// rolling 15 minutes before that ticket is blocked for an hour. 0 disables the cap.
	WriteCapPerTicket int `json:"write_cap_per_ticket"`

	// OpsRoomID is the Webex recipient operator alerts go to: intake failures, write-cap blocks
	// and webhook staleness. nil logs them instead.
	OpsRoomID *int `json:"ops_room_id"`

	// RedirectRoomID, when set, receives every notification instead of its intended recipient,
	// prefixed with who it was for. It is the parallel-run switch: workflows keep their real
	// targets and nobody else sees the messages. While set, notifications are sent even under
	// dry run, because the redirect is the safety.
	RedirectRoomID *int `json:"redirect_room_id"`

	// StaleAlertMinutes is how long business hours (07:30 to 19:00 America/Chicago, weekdays) may
	// pass with no ticket webhook before the ops room is told. 0 disables the check.
	StaleAlertMinutes int `json:"stale_alert_minutes"`

	// HistoryRetentionDays is how long workflow run summaries and ticket history events are kept.
	// They are the same story told twice, so they age out together. 0 keeps them forever.
	HistoryRetentionDays int `json:"history_retention_days"`

	// Business hours bound the stale-webhook alert: silence outside them is normal. Times are
	// HH:MM in BusinessZone; BusinessDays is a comma-separated list of mon..sun.
	BusinessOpen  string `json:"business_open"`
	BusinessClose string `json:"business_close"`
	BusinessDays  string `json:"business_days"`
	BusinessZone  string `json:"business_zone"`

	// MCPEnabled switches on the MCP server and its OAuth endpoints. Off, every one of them
	// answers 404; grants and tokens already issued are kept for when it comes back.
	MCPEnabled bool `json:"mcp_enabled"`
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
	WriteCapPerTicket       *int    `json:"write_cap_per_ticket"`
	// OpsRoomID and RedirectRoomID clear the setting when sent as 0.
	OpsRoomID            *int    `json:"ops_room_id"`
	RedirectRoomID       *int    `json:"redirect_room_id"`
	StaleAlertMinutes    *int    `json:"stale_alert_minutes"`
	HistoryRetentionDays *int    `json:"history_retention_days"`
	BusinessOpen         *string `json:"business_open"`
	BusinessClose        *string `json:"business_close"`
	BusinessDays         *string `json:"business_days"`
	BusinessZone         *string `json:"business_zone"`
	MCPEnabled           *bool   `json:"mcp_enabled"`
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
	WriteCapPerTicket:       20,
	StaleAlertMinutes:       60,
	HistoryRetentionDays:    90,
	BusinessOpen:            "07:30",
	BusinessClose:           "19:00",
	BusinessDays:            "mon,tue,wed,thu,fri",
	BusinessZone:            "America/Chicago",
	MCPEnabled:              false,
}
