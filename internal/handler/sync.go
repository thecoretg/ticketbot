package handler

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type SyncHandler struct {
	CW    *cwsvc.Service
	Webex *webexsvc.Service
}

func NewSyncHandler(cw *cwsvc.Service, wx *webexsvc.Service) *SyncHandler {
	return &SyncHandler{
		CW:    cw,
		Webex: wx,
	}
}

func (h *SyncHandler) HandleSync(c *gin.Context) {
	p := &struct {
		WebexRooms         bool  `json:"webex_rooms"`
		CWBoards           bool  `json:"cw_boards"`
		CWTickets          bool  `json:"cw_tickets"`
		BoardIDs           []int `json:"board_ids"`
		MaxConcurrentSyncs int   `json:"max_concurrent_syncs"`
	}{}

	if err := c.ShouldBindJSON(p); err != nil {
		c.Error(fmt.Errorf("bad json payload: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "sync started"})

	go func() {
		ctx := context.Background()
		if p.WebexRooms {
			go h.syncRooms(ctx)
		}

		if p.CWBoards {
			go h.syncBoards(ctx)
		}

		if p.CWTickets {
			go h.syncTickets(ctx, p.BoardIDs, p.MaxConcurrentSyncs)
		}
	}()
}

func (h *SyncHandler) syncRooms(ctx context.Context) {
	if err := h.CW.SyncBoards(ctx); err != nil {
		slog.Error("syncing connectwise boards", "error", err)
	}
}

func (h *SyncHandler) syncBoards(ctx context.Context) {
	if err := h.CW.SyncBoards(ctx); err != nil {
		slog.Error("syncing connectwise boards", "error", err)
	}
}

func (h *SyncHandler) syncTickets(ctx context.Context, boardIDs []int, maxConcurrent int) {
	if maxConcurrent == 0 || maxConcurrent > 10 {
		maxConcurrent = 5
	}

	if err := h.CW.SyncOpenTickets(ctx, boardIDs, maxConcurrent); err != nil {
		slog.Error("syncing connectwise tickets", "error", err)
	}
}
