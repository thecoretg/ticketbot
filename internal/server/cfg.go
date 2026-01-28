package server

import (
	"log/slog"
	"os"
	"strconv"

	"github.com/thecoretg/ticketbot/internal/models"
)

// loadConfigOverrides takes any explicitly set config values from env variables
// and sets them in the config. These are overridden if set via the config routes;
// this is primarily for testing purposes.
func loadConfigOverrides(current *models.Config) *models.Config {
	var (
		attemptNotify   *bool
		skipLaunchSyncs *bool
		maxLen          *int
		maxConSyncs     *int
	)

	switch os.Getenv("ATTEMPT_NOTIFY") {
	case "true":
		v := true
		attemptNotify = &v
	case "false":
		v := false
		attemptNotify = &v
	}

	switch os.Getenv("SKIP_LAUNCH_SYNCS") {
	case "true":
		v := true
		skipLaunchSyncs = &v
	case "false":
		v := false
		skipLaunchSyncs = &v
	}

	mlInt, err := strconv.Atoi(os.Getenv("MAX_MSG_LENGTH"))
	if err == nil {
		v := mlInt
		maxLen = &v
	}

	msInt, err := strconv.Atoi(os.Getenv("MAX_CONCURRENT_SYNCS"))
	if err == nil {
		v := msInt
		maxConSyncs = &v
	}

	if skipLaunchSyncs != nil {
		slog.Info("SKIP_LAUNCH_SYNCS overridden via env", "value", *skipLaunchSyncs)
		current.SkipLaunchSyncs = *skipLaunchSyncs
	}

	if attemptNotify != nil {
		slog.Info("ATTEMPT_NOTIFY overridden via env", "value", *attemptNotify)
		current.AttemptNotify = *attemptNotify
	}

	if maxLen != nil {
		slog.Info("MAX_MSG_LENGTH overridden via env", "value", *maxLen)
		current.MaxMessageLength = *maxLen
	}

	if maxConSyncs != nil {
		slog.Info("MAX_CONCURRENT_SYNCS overridden via env", "value", *maxConSyncs)
		current.MaxConcurrentSyncs = *maxConSyncs
	}

	return current
}
