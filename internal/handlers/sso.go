package handlers

import (
	"errors"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/service/sso"
	"github.com/thecoretg/ticketbot/models"
)

type SSOHandler struct {
	svc *sso.Service
}

func NewSSOHandler(svc *sso.Service) *SSOHandler {
	return &SSOHandler{svc: svc}
}

// Methods is unauthenticated: the login card asks it which sign-in options to show.
func (h *SSOHandler) Methods(w http.ResponseWriter, _ *http.Request) {
	outputJSON(w, h.svc.Methods())
}

func (h *SSOHandler) Status(w http.ResponseWriter, r *http.Request) {
	st, err := h.svc.Status(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}
	outputJSON(w, st)
}

// Test runs discovery and a client-credentials grant. Failures are reported in the body with a
// 200 so the dashboard can show the message inline.
func (h *SSOHandler) Test(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.TestConnection(r.Context()); err != nil {
		outputJSON(w, M{"ok": false, "error": err.Error()})
		return
	}
	outputJSON(w, M{"ok": true})
}

type mappingRequest struct {
	EntraRole string      `json:"entra_role"`
	Role      models.Role `json:"role"`
}

func (h *SSOHandler) UpsertMapping(w http.ResponseWriter, r *http.Request) {
	var p mappingRequest
	if err := decodeJSON(r, &p); err != nil {
		badPayloadError(w, err)
		return
	}
	m, err := h.svc.UpsertMapping(r.Context(), p.EntraRole, p.Role)
	if err != nil {
		if errors.Is(err, sso.ErrEmptyRole) || errors.Is(err, models.ErrInvalidRole) {
			writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
			return
		}
		internalServerError(w, err)
		return
	}
	outputJSON(w, m)
}

func (h *SSOHandler) DeleteMapping(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	if err := h.svc.DeleteMapping(r.Context(), id); err != nil {
		internalServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
