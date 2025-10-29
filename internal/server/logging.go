package server

import (
	"log/slog"
	"os"
)

// setInitialLogger sets the default logger and is run at server startup
func setInitialLogger() {
	j := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	slog.SetDefault(slog.New(j))
}

// setLogLevel is used for changing the log level outside of server startup
func setLogLevel(debug bool) {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	slog.Info("setting debug level", "debug", debug)
	slog.SetLogLoggerLevel(level)
}
