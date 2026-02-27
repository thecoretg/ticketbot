package models

import "errors"

var ErrConfigNotFound = errors.New("config not found")

type Config struct {
	// The app will only ever have one config in the config table, so this will always just be 1.
	ID int `json:"id"`

	// AttemptNotify is a full killswitch for attempting to process notifications for tickets. If it is off,
	// all notifier rules will be disregarded.
	AttemptNotify bool `json:"attempt_notify"`

	// MaxMessageLength is the max amount of characters in a notification's ticket note output before
	// it truncates and adds a "..." to the end.
	MaxMessageLength int `json:"max_message_length"`

	// MaxConcurrentSyncs is the maximum amount of syncs that can be run at once. This defaults to 5
	// as it has been deemed a good amount to prevent constant rate limits from Connectwise.
	MaxConcurrentSyncs int `json:"max_concurrent_syncs"`

	// SkipLaunchSyncs is a flag to skip the automatic syncing of webex recipients and connectwise boards.
	SkipLaunchSyncs bool `json:"skip_launch_syncs"`
}

var DefaultConfig = Config{
	ID:                 1,
	AttemptNotify:      false,
	MaxMessageLength:   300,
	MaxConcurrentSyncs: 5,
	SkipLaunchSyncs:    false,
}
