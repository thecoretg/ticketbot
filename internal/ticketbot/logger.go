package ticketbot

import (
	"fmt"
	"log/slog"
	"os"
)

func setLogger(lvl string, toFile bool) error {
	level := slog.LevelInfo
	if lvl == "1" || lvl == "true" {
		level = slog.LevelDebug
	}

	var err error
	handler := newStdoutHandler(level)
	if toFile {
		handler, err = newFileHandler("ticketbot.log", level)
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

func newFileHandler(filePath string, level slog.Level) (*slog.TextHandler, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0666)
	if err != nil {
		return nil, fmt.Errorf("opening log file %s: %w", filePath, err)
	}
	
	return slog.NewTextHandler(file, &slog.HandlerOptions{
		Level: level,
	}), nil
}

func newStdoutHandler(level slog.Level) *slog.TextHandler {
	return slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})
}
