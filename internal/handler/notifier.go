package handler

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotifierHandler struct {
	BoardRepo    models.BoardRepository
	RoomRepo     models.WebexRoomRepository
	NotifierRepo models.NotifierRepository
}

func NewNotifierHandler(r models.NotifierRepository, br models.BoardRepository, wr models.WebexRoomRepository) *NotifierHandler {
	return &NotifierHandler{
		BoardRepo:    br,
		RoomRepo:     wr,
		NotifierRepo: r,
	}
}

func (h *NotifierHandler) ListNotifiers(c *gin.Context) {
	n, err := h.NotifierRepo.ListAll(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, n)
}

func (h *NotifierHandler) AddNotifier(c *gin.Context) {
	ctx := c.Request.Context()
	p := &models.Notifier{}
	if err := c.ShouldBindJSON(p); err != nil {
		c.JSON(http.StatusBadRequest, errorOutput(fmt.Errorf("bad json payload: %w", err)))
		return
	}

	exists, err := h.NotifierRepo.Exists(ctx, p.CwBoardID, p.WebexRoomID)
	if err != nil {
		c.Error(err)
		return
	}

	if exists {
		err = fmt.Errorf("notifier with board id %d and room id %d already exists", p.CwBoardID, p.WebexRoomID)
		c.JSON(http.StatusConflict, errorOutput(err))
	}

	if _, err = h.BoardRepo.Get(ctx, p.CwBoardID); err != nil {
		if errors.Is(err, models.ErrBoardNotFound) {
			c.JSON(http.StatusNotFound, err)
			return
		}
		c.Error(err)
		return
	}

	if _, err = h.RoomRepo.Get(ctx, p.WebexRoomID); err != nil {
		if errors.Is(err, models.ErrWebexRoomNotFound) {
			c.JSON(http.StatusNotFound, err)
			return
		}
		c.Error(err)
		return
	}

	n, err := h.NotifierRepo.Insert(c.Request.Context(), p)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, n)
}
