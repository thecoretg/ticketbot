package handlers

import (
	"log/slog"
	"net/http"
	"time"
)

type AdminHandler struct {
	shutdown func()
}

func NewAdminHandler(shutdown func()) *AdminHandler {
	return &AdminHandler{shutdown: shutdown}
}

func (h *AdminHandler) HandleRestart(w http.ResponseWriter, r *http.Request) {
	slog.Info("restart requested via web panel")
	w.WriteHeader(http.StatusNoContent)

	// trigger shutdown after the response is sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.shutdown()
	}()
}
