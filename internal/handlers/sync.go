package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/models"
)

type SyncHandler struct {
	Svc  *syncsvc.Service
	Jobs repos.ScheduledJobRepository
	cfg  *models.Config
}

func NewSyncHandler(svc *syncsvc.Service, jobs repos.ScheduledJobRepository, cfg *models.Config) *SyncHandler {
	return &SyncHandler{Svc: svc, Jobs: jobs, cfg: cfg}
}

func (h *SyncHandler) HandleSyncStatus(w http.ResponseWriter, r *http.Request) {
	st := h.Svc.Status()
	nightly, err := h.Jobs.Get(r.Context(), syncsvc.NightlyJobName)
	if err != nil {
		internalServerError(w, err)
		return
	}
	st.Nightly = nightly
	writeJSON(w, 200, st)
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
