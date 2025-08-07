package ticketbot

import (
	"fmt"
	"log/slog"
	"os"
)

func setLogger(debug, toFile bool, logFilePath string) error {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	var err error
	handler := newStdoutHandler(level)
	if toFile {
		if logFilePath == "" {
			logFilePath = "ticketbot.log"
		}

		handler, err = newFileHandler(logFilePath, level)
		if err != nil {
			return fmt.Errorf("creating file handler: %w", err)
		}
	}

	// this only shows if debug is enabled
	// you can tell by the way that it is
	slog.Debug("debug enabled")
	slog.SetDefault(slog.New(handler))
	return nil
}

func newFileHandler(filePath string, level slog.Level) (*slog.JSONHandler, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", filePath, err)
	}

	return slog.NewJSONHandler(file, &slog.HandlerOptions{
		Level: level,
	}), nil
}

func newStdoutHandler(level slog.Level) *slog.JSONHandler {
	return slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
}
