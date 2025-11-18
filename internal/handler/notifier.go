package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotifierHandler struct {
	Repo models.NotifierRepository
}

func NewNotifierHandler(r models.NotifierRepository) *NotifierHandler {
	return &NotifierHandler{Repo: r}
}

func (h *NotifierHandler) ListNotifiers(c *gin.Context) {
	n, err := h.Repo.ListAll(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, n)
}

func (h *NotifierHandler) AddNotifier(c *gin.Context) {
	p := &models.Notifier{}
	if err := c.ShouldBindJSON(p); err != nil {
		c.JSON(http.StatusBadRequest, errorOutput(fmt.Errorf("bad json payload: %w", err)))
		return
	}

	exists, err := h.Repo.Exists(c.Request.Context(), p.CwBoardID, p.WebexRoomID)
	if err != nil {
		c.Error(err)
		return
	}

	if exists {
		err = fmt.Errorf("notifier with board id %d and room id %d already exists", p.CwBoardID, p.WebexRoomID)
		c.JSON(http.StatusConflict, errorOutput(err))
	}

	// TODO: board existence check?

	n, err := h.Repo.Insert(c.Request.Context(), p)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, n)
}
