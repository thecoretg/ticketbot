package handlers

import (
	"errors"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/models"
)

type NotifierHandler struct {
	Svc notifier.Service
}

func NewNotifierHandler(svc *notifier.Service) *NotifierHandler {
	return &NotifierHandler{
		Svc: *svc,
	}
}

func (h *NotifierHandler) ListForwards(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	filter := r.URL.Query().Get("filter")

	var n []*models.NotifierForwardFull
	var err error

	switch filter {
	case "active":
		n, err = h.Svc.ListForwardsActive(ctx)
	case "inactive":
		n, err = h.Svc.ListForwardsInactive(ctx)
	case "not-expired":
		n, err = h.Svc.ListForwardsNotExpired(ctx)
	default:
		n, err = h.Svc.ListForwardsFull(ctx)
	}

	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, n)
}

func (h *NotifierHandler) GetForward(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	f, err := h.Svc.GetForward(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, f)
}

func (h *NotifierHandler) AddUserForward(w http.ResponseWriter, r *http.Request) {
	p := &models.NotifierForward{}
	if err := decodeJSON(r, p); err != nil {
		badPayloadError(w, err)
		return
	}

	f, err := h.Svc.AddForward(r.Context(), p)
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, f)
}

func (h *NotifierHandler) UpdateUserForward(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	p := &models.NotifierForward{}
	if err := decodeJSON(r, p); err != nil {
		badPayloadError(w, err)
		return
	}

	f, err := h.Svc.UpdateForward(r.Context(), id, p)
	if err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, f)
}

func (h *NotifierHandler) DeleteUserForward(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	if err := h.Svc.DeleteForward(r.Context(), id); err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
