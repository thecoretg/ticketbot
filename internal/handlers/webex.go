package handlers

import (
	"errors"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/models"
)

type WebexHandler struct {
	WebexSvc *webexsvc.Service
}

func NewWebexHandler(wx *webexsvc.Service) *WebexHandler {
	return &WebexHandler{
		WebexSvc: wx,
	}
}

func (h *WebexHandler) ListRecipients(w http.ResponseWriter, r *http.Request) {
	recips, err := h.WebexSvc.ListRecipients(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, recips)
}

func (h *WebexHandler) GetRoom(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	recip, err := h.WebexSvc.GetRecipient(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrWebexRecipientNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, recip)
}
