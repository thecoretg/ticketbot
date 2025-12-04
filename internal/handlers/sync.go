package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
)

type SyncHandler struct {
	Svc *syncsvc.Service
}

func NewSyncHandler(svc *syncsvc.Service) *SyncHandler {
	return &SyncHandler{
		Svc: svc,
	}
}

func (h *SyncHandler) HandleSync(c *gin.Context) {
	p := &models.SyncPayload{}

	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	ctx := context.WithoutCancel(c.Request.Context())
	if p.WebexRecipients {
		go h.syncWebexRecipients(ctx, p.MaxConcurrentSyncs)
	}

	if p.CWBoards {
		go h.syncBoards(ctx)
	}

	if p.CWTickets {
		go h.syncTickets(ctx, p.BoardIDs, p.MaxConcurrentSyncs)
	}

	resultJSON(c, "sync started")
}

func (h *SyncHandler) syncWebexRecipients(ctx context.Context, maxConcurrent int) {
	if err := h.Svc.SyncWebexRecipients(ctx, maxConcurrent); err != nil {
		slog.Error("syncing webex rooms", "error", err)
	}
}

func (h *SyncHandler) syncBoards(ctx context.Context) {
	if err := h.Svc.SyncBoards(ctx); err != nil {
		slog.Error("syncing connectwise boards", "error", err)
	}
}

func (h *SyncHandler) syncTickets(ctx context.Context, boardIDs []int, maxConcurrent int) {
	if maxConcurrent == 0 || maxConcurrent > 10 {
		maxConcurrent = 5
	}

	if err := h.Svc.SyncOpenTickets(ctx, boardIDs, maxConcurrent); err != nil {
		slog.Error("syncing connectwise tickets", "error", err)
	}
}
