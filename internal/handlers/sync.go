package handlers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/models"
)

type SyncHandler struct {
	Svc *syncsvc.Service
	cfg *models.Config
}

func NewSyncHandler(svc *syncsvc.Service, cfg *models.Config) *SyncHandler {
	return &SyncHandler{Svc: svc, cfg: cfg}
}

func (h *SyncHandler) HandleSyncStatus(w http.ResponseWriter, r *http.Request) {
	status := &models.SyncStatusResponse{Status: h.Svc.IsSyncing()}
	writeJSON(w, 200, status)
}

func (h *SyncHandler) HandleSync(w http.ResponseWriter, r *http.Request) {
	p := &models.SyncPayload{}

	if err := decodeJSON(r, p); err != nil {
		badPayloadError(w, err)
		return
	}

	if p.MaxConcurrentSyncs == 0 {
		p.MaxConcurrentSyncs = h.cfg.MaxConcurrentSyncs
	}

	ctx := context.WithoutCancel(r.Context())
	go func() {
		if err := h.Svc.Sync(ctx, p); err != nil {
			slog.Error("syncing", "error", err.Error())
		}
	}()

	resultJSON(w, "sync started")
}
