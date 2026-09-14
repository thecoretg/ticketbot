package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/models"
)

type TicketsHandler struct {
	Service *cwsvc.Service
}

func NewTicketsHandler(svc *cwsvc.Service) *TicketsHandler {
	return &TicketsHandler{Service: svc}
}

// List handles GET /tickets?board_id=&status_id=&closed=&deleted=&q=&page=&page_size=
func (h *TicketsHandler) List(c *gin.Context) {
	f := models.TicketFilter{Search: c.Query("q")}

	var err error
	if f.BoardID, err = optionalIntQuery(c, "board_id"); err != nil {
		badQueryError(c, err)
		return
	}
	if f.StatusID, err = optionalIntQuery(c, "status_id"); err != nil {
		badQueryError(c, err)
		return
	}
	if f.Closed, err = optionalBoolQuery(c, "closed"); err != nil {
		badQueryError(c, err)
		return
	}
	if v, err := optionalBoolQuery(c, "deleted"); err != nil {
		badQueryError(c, err)
		return
	} else if v != nil {
		f.IncludeDeleted = *v
	}
	if f.Page, err = intQueryDefault(c, "page", 1); err != nil {
		badQueryError(c, err)
		return
	}
	if f.PageSize, err = intQueryDefault(c, "page_size", 0); err != nil {
		badQueryError(c, err)
		return
	}

	page, err := h.Service.ListTickets(c.Request.Context(), f)
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, page)
}

// Get handles GET /tickets/:id
func (h *TicketsHandler) Get(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	d, err := h.Service.GetTicketDetail(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, d)
}

// Raw handles GET /tickets/:id/raw
func (h *TicketsHandler) Raw(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	raw, err := h.Service.GetTicketRaw(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	if len(raw) == 0 {
		notFoundError(c, fmt.Errorf("no raw ticket data stored for ticket %d", id))
		return
	}

	c.Data(http.StatusOK, "application/json", raw)
}

// Events handles GET /tickets/:id/events?limit=&before_id=
func (h *TicketsHandler) Events(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	limit, err := intQueryDefault(c, "limit", 100)
	if err != nil {
		badQueryError(c, err)
		return
	}

	var beforeID *int64
	if s := c.Query("before_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			badQueryError(c, fmt.Errorf("before_id: %w", err))
			return
		}
		beforeID = &v
	}

	events, err := h.Service.ListTicketEvents(c.Request.Context(), id, limit, beforeID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	if events == nil {
		events = []*models.TicketEvent{}
	}
	outputJSON(c, events)
}

func optionalIntQuery(c *gin.Context, key string) (*int, error) {
	s := c.Query(key)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &v, nil
}

func optionalBoolQuery(c *gin.Context, key string) (*bool, error) {
	s := c.Query(key)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &v, nil
}

func intQueryDefault(c *gin.Context, key string, def int) (int, error) {
	s := c.Query(key)
	if s == "" {
		return def, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}
