package middleware

import (
	"log/slog"
	"strings"
)

type SlogWriter struct {
	logger *slog.Logger
}

func NewSlogWriter(logger *slog.Logger) *SlogWriter {
	return &SlogWriter{logger: logger}
}

func (w *SlogWriter) Write(p []byte) (n int, err error) {
	w.logger.Info(strings.TrimSpace(string(p)))
	return len(p), nil
}
