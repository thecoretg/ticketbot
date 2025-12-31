package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type WebexHandler struct {
	WebexSvc *webexsvc.Service
}

func NewWebexHandler(wx *webexsvc.Service) *WebexHandler {
	return &WebexHandler{
		WebexSvc: wx,
	}
}

func (h *WebexHandler) ListRecipients(c *gin.Context) {
	r, err := h.WebexSvc.ListRecipients(c.Request.Context())
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

	r, err := h.WebexSvc.GetRecipient(c.Request.Context(), id)
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
