package logging

import (
	"log/slog"
	"os"
)

func NewDefaultLogger(level slog.Leveler) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	}))
}
