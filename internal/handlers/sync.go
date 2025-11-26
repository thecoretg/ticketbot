package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
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
	p := &models.SyncPayload{}

	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	ctx := context.WithoutCancel(c.Request.Context())
	if p.WebexRooms {
		go h.syncRooms(ctx)
	}

	if p.CWBoards {
		go h.syncBoards(ctx)
	}

	if p.CWTickets {
		go h.syncTickets(ctx, p.BoardIDs, p.MaxConcurrentSyncs)
	}

	resultJSON(c, "sync started")
}

func (h *SyncHandler) syncRooms(ctx context.Context) {
	if err := h.Webex.SyncRooms(ctx); err != nil {
		slog.Error("syncing webex rooms", "error", err)
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
