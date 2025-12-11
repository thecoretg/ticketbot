package handlers

import (
	"errors"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type WebexHandler struct {
	Service *webexsvc.Service
}

func NewWebexHandler(svc *webexsvc.Service) *WebexHandler {
	return &WebexHandler{Service: svc}
}

func (h *WebexHandler) HandleMessages(c *gin.Context) {
	w := &webex.MessageHookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		badPayloadError(c, err)
		return
	}
	id := w.Data.ID

	msg, err := h.Service.GetMessage(c.Request.Context(), id)
	if err != nil {
		internalServerError(c, err)
		return
	}

	slog.Info("received message from webex", "id", msg.ID, "sender", msg.PersonEmail, "text", msg.Text)
	resultJSON(c, "received webex message")
}

func (h *WebexHandler) ListRecipients(c *gin.Context) {
	r, err := h.Service.ListRecipient(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}

func (h *WebexHandler) GetRoom(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	r, err := h.Service.GetRecipient(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrWebexRecipientNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}
