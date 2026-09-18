package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

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
func (h *TicketsHandler) List(w http.ResponseWriter, r *http.Request) {
	f := models.TicketFilter{Search: r.URL.Query().Get("q")}

	var err error
	if f.BoardID, err = optionalIntQuery(r, "board_id"); err != nil {
		badQueryError(w, err)
		return
	}
	if f.StatusID, err = optionalIntQuery(r, "status_id"); err != nil {
		badQueryError(w, err)
		return
	}
	if f.Closed, err = optionalBoolQuery(r, "closed"); err != nil {
		badQueryError(w, err)
		return
	}
	if v, err := optionalBoolQuery(r, "deleted"); err != nil {
		badQueryError(w, err)
		return
	} else if v != nil {
		f.IncludeDeleted = *v
	}
	if f.Page, err = intQueryDefault(r, "page", 1); err != nil {
		badQueryError(w, err)
		return
	}
	if f.PageSize, err = intQueryDefault(r, "page_size", 0); err != nil {
		badQueryError(w, err)
		return
	}

	page, err := h.Service.ListTickets(r.Context(), f)
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, page)
}

// Get handles GET /tickets/:id
func (h *TicketsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	d, err := h.Service.GetTicketDetail(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, d)
}

// Raw handles GET /tickets/:id/raw
func (h *TicketsHandler) Raw(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	raw, err := h.Service.GetTicketRaw(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	if len(raw) == 0 {
		notFoundError(w, fmt.Errorf("no raw ticket data stored for ticket %d", id))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

// Events handles GET /tickets/:id/events?limit=&before_id=
func (h *TicketsHandler) Events(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	limit, err := intQueryDefault(r, "limit", 100)
	if err != nil {
		badQueryError(w, err)
		return
	}

	var beforeID *int64
	if s := r.URL.Query().Get("before_id"); s != "" {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			badQueryError(w, fmt.Errorf("before_id: %w", err))
			return
		}
		beforeID = &v
	}

	events, err := h.Service.ListTicketEvents(r.Context(), id, limit, beforeID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	if events == nil {
		events = []*models.TicketEvent{}
	}
	outputJSON(w, events)
}

func optionalIntQuery(r *http.Request, key string) (*int, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &v, nil
}

func optionalBoolQuery(r *http.Request, key string) (*bool, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return nil, nil
	}
	v, err := strconv.ParseBool(s)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", key, err)
	}
	return &v, nil
}

func intQueryDefault(r *http.Request, key string, def int) (int, error) {
	s := r.URL.Query().Get(key)
	if s == "" {
		return def, nil
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, err)
	}
	return v, nil
}
