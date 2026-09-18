package handlers

import (
	"github.com/thecoretg/ticketbot/internal/logging"
	"net/http"
)

type LogsHandler struct {
	buf *logging.BufferHandler
}

func NewLogsHandler(buf *logging.BufferHandler) *LogsHandler {
	return &LogsHandler{buf: buf}
}

func (h *LogsHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	outputJSON(w, h.buf.Entries())
}
