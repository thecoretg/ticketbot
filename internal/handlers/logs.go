package handlers

import (
	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/logging"
)

type LogsHandler struct {
	buf *logging.BufferHandler
}

func NewLogsHandler(buf *logging.BufferHandler) *LogsHandler {
	return &LogsHandler{buf: buf}
}

func (h *LogsHandler) HandleList(c *gin.Context) {
	outputJSON(c, h.buf.Entries())
}
