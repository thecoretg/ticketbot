package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/models"
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

func (h *SyncHandler) HandleSyncStatus(c *gin.Context) {
	status := &models.SyncStatusResponse{Status: h.Svc.IsSyncing()}
	c.JSON(200, status)
}

func (h *SyncHandler) HandleSync(c *gin.Context) {
	p := &models.SyncPayload{}

	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	ctx := context.WithoutCancel(c.Request.Context())
	go func() {
		if err := h.Svc.Sync(ctx, p); err != nil {
			slog.Error("syncing", "error", err.Error())
		}
	}()

	resultJSON(c, "sync started")
}
