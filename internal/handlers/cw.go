package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
)

type CWHandler struct {
	Service *cwsvc.Service
}

func NewCWHandler(svc *cwsvc.Service) *CWHandler {
	return &CWHandler{Service: svc}
}

func (h *CWHandler) ListBoards(c *gin.Context) {
	b, err := h.Service.ListBoards(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, b)
}

func (h *CWHandler) GetBoard(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	b, err := h.Service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrBoardNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, b)
}
