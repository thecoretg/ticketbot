package handlers

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminHandler struct {
	shutdown func()
}

func NewAdminHandler(shutdown func()) *AdminHandler {
	return &AdminHandler{shutdown: shutdown}
}

func (h *AdminHandler) HandleRestart(c *gin.Context) {
	slog.Info("restart requested via web panel")
	c.Status(http.StatusNoContent)

	// trigger shutdown after the response is sent
	go func() {
		time.Sleep(100 * time.Millisecond)
		h.shutdown()
	}()
}
