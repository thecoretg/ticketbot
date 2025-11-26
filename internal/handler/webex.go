package handler

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type WebexHandler struct {
	Service *webexsvc.Service
}

func NewWebexHandler(svc *webexsvc.Service) *WebexHandler {
	return &WebexHandler{Service: svc}
}

func (h *WebexHandler) ListRooms(c *gin.Context) {
	r, err := h.Service.ListRooms(c.Request.Context())
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

	r, err := h.Service.GetRoom(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrWebexRoomNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}
