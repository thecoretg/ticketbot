package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/journal"
	"github.com/thecoretg/ticketbot/models"
)

type TicketJournalHandler struct {
	Svc *journal.Service
}

func NewTicketJournalHandler(svc *journal.Service) *TicketJournalHandler {
	return &TicketJournalHandler{Svc: svc}
}

// List returns the ticket journal overview (snapshot columns, no run timelines).
func (h *TicketJournalHandler) List(c *gin.Context) {
	items, err := h.Svc.ListSummaries(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}
	outputJSON(c, items)
}

// Get returns a single ticket's full journal including its run timeline.
func (h *TicketJournalHandler) Get(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	j, err := h.Svc.Get(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTicketJournalNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}
	outputJSON(c, j)
}
