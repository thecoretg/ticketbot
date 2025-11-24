package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

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

func (h *CWHandler) SyncOpenTickets(c *gin.Context) {
	p := &struct {
		BoardIDs           []int `json:"board_ids"`
		MaxConcurrentSyncs int   `json:"max_concurrent_syncs"`
	}{}

	if err := c.ShouldBindJSON(p); err != nil {
		c.Error(fmt.Errorf("bad json payload: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "ticket sync started"})

	go func() {
		if p.MaxConcurrentSyncs == 0 {
			p.MaxConcurrentSyncs = 5
		}

		if err := h.Service.SyncOpenTickets(c.Request.Context(), p.BoardIDs, p.MaxConcurrentSyncs); err != nil {
			slog.Error("syncing connectwise tickets", "error", err)
		}
	}()
}

func (h *CWHandler) ListBoards(c *gin.Context) {
	b, err := h.Service.ListBoards(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, b)
}

func (h *CWHandler) GetBoard(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, badIntErrorOutput(c.Param("id")))
		return
	}

	b, err := h.Service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrBoardNotFound) {
			c.JSON(http.StatusNotFound, errorOutput(err))
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, b)
}

func (h *CWHandler) SyncBoards(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"result": "board sync started"})

	go func() {
		if err := h.Service.SyncBoards(context.Background()); err != nil {
			slog.Error("syncing connectwise boards", "err", err)
		}
	}()
}
