package handlers

import (
	"errors"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/models"
)

type CWHandler struct {
	Service *cwsvc.Service
}

func NewCWHandler(svc *cwsvc.Service) *CWHandler {
	return &CWHandler{Service: svc}
}

func (h *CWHandler) ListBoards(c *gin.Context) {
	b, err := h.Service.ListBoards(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, b)
}

func (h *CWHandler) ListMembers(c *gin.Context) {
	m, err := h.Service.ListMembers(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, m)
}

// ListCompanies live-searches CW companies by name (?q=). Used by the workflow
// company picker.
func (h *CWHandler) ListCompanies(c *gin.Context) {
	companies, err := h.Service.SearchCompanies(c.Request.Context(), c.Query("q"))
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, companies)
}

// ListContacts live-searches CW contacts within a company (?company=<identifier>&q=).
func (h *CWHandler) ListContacts(c *gin.Context) {
	contacts, err := h.Service.SearchContacts(c.Request.Context(), c.Query("company"), c.Query("q"))
	if err != nil {
		badPayloadError(c, err)
		return
	}

	outputJSON(c, contacts)
}

// ListBoardStatuses live-fetches the active statuses for a board (?q= filters by name).
func (h *CWHandler) ListBoardStatuses(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	statuses, err := h.Service.LiveBoardStatuses(c.Request.Context(), id, c.Query("q"))
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, statuses)
}

// ListBoardTypes lists a board's active ticket types (?q= filters by name).
func (h *CWHandler) ListBoardTypes(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	types, err := h.Service.ListBoardTypes(c.Request.Context(), id, c.Query("q"))
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, types)
}

// ListBoardSubTypes lists a board's active ticket subtypes (?q= filters by name).
func (h *CWHandler) ListBoardSubTypes(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	subtypes, err := h.Service.ListBoardSubTypes(c.Request.Context(), id, c.Query("q"))
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, subtypes)
}

func (h *CWHandler) GetBoard(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	b, err := h.Service.GetBoard(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrBoardNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, b)
}
