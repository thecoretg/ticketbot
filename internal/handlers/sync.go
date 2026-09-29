package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/middleware"
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
	writeJSON(w, 200, h.Svc.Status())
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

	startedBy := ""
	if u := middleware.User(r.Context()); u != nil {
		startedBy = u.EmailAddress
	}

	// the sync outlives this request
	ctx := context.WithoutCancel(r.Context())
	if err := h.Svc.Start(ctx, p, startedBy); err != nil {
		if errors.Is(err, syncsvc.ErrSyncRunning) {
			conflictError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	resultJSON(w, "sync started")
}

func (h *SyncHandler) HandleSyncCancel(w http.ResponseWriter, r *http.Request) {
	by := ""
	if u := middleware.User(r.Context()); u != nil {
		by = u.EmailAddress
	}

	if err := h.Svc.Cancel(by); err != nil {
		if errors.Is(err, syncsvc.ErrNoSyncRunning) {
			conflictError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	resultJSON(w, "sync cancelling")
}
