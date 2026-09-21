package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/thecoretg/ticketbot/internal/service/intake"
	"github.com/thecoretg/ticketbot/models"
)

// IntakeHandler is the admin view of the webhook queue.
type IntakeHandler struct {
	Svc *intake.Service
}

func NewIntakeHandler(svc *intake.Service) *IntakeHandler {
	return &IntakeHandler{Svc: svc}
}

// List handles GET /intake?status=failed&limit=100.
func (h *IntakeHandler) List(w http.ResponseWriter, r *http.Request) {
	var status *models.IntakeStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := models.IntakeStatus(s)
		status = &st
	}
	limit, err := intQueryDefault(r, "limit", 100)
	if err != nil {
		badQueryError(w, err)
		return
	}
	rows, err := h.Svc.List(r.Context(), status, limit)
	if err != nil {
		internalServerError(w, err)
		return
	}
	outputJSON(w, rows)
}

func (h *IntakeHandler) Stats(w http.ResponseWriter, r *http.Request) {
	st, err := h.Svc.Stats(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}
	outputJSON(w, st)
}

// Hourly handles GET /intake/hourly?days=7.
func (h *IntakeHandler) Hourly(w http.ResponseWriter, r *http.Request) {
	days, err := intQueryDefault(r, "days", 7)
	if err != nil {
		badQueryError(w, err)
		return
	}
	rows, err := h.Svc.Hourly(r.Context(), days)
	if err != nil {
		internalServerError(w, err)
		return
	}
	outputJSON(w, rows)
}

func (h *IntakeHandler) Retry(w http.ResponseWriter, r *http.Request) {
	h.act(w, r, h.Svc.Retry)
}

func (h *IntakeHandler) Discard(w http.ResponseWriter, r *http.Request) {
	h.act(w, r, h.Svc.Discard)
}

func (h *IntakeHandler) act(w http.ResponseWriter, r *http.Request, fn func(context.Context, int64) (*models.WebhookIntake, error)) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		errJSON(w, http.StatusBadRequest, fmt.Errorf("%s is not a valid id", r.PathValue("id")))
		return
	}
	row, err := fn(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrIntakeNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}
	outputJSON(w, row)
}
