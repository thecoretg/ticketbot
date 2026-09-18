package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/models"
)

type CWHandler struct {
	Service *cwsvc.Service
}

func NewCWHandler(svc *cwsvc.Service) *CWHandler {
	return &CWHandler{Service: svc}
}

func (h *CWHandler) ListBoards(w http.ResponseWriter, r *http.Request) {
	b, err := h.Service.ListBoards(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, b)
}

func (h *CWHandler) ListMembers(w http.ResponseWriter, r *http.Request) {
	m, err := h.Service.ListMembers(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, m)
}

// ListCompanies handles GET /cw/companies?q=&ids=1,2&limit=
func (h *CWHandler) ListCompanies(w http.ResponseWriter, r *http.Request) {
	ids, err := idListQuery(r, "ids")
	if err != nil {
		badQueryError(w, err)
		return
	}
	limit, err := intQueryDefault(r, "limit", 0)
	if err != nil {
		badQueryError(w, err)
		return
	}

	out, err := h.Service.SearchCompanies(r.Context(), models.CompanySearch{Query: r.URL.Query().Get("q"), IDs: ids, Limit: limit})
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, out)
}

// ListContacts handles GET /cw/contacts?q=&company_id=&ids=1,2&limit=
func (h *CWHandler) ListContacts(w http.ResponseWriter, r *http.Request) {
	ids, err := idListQuery(r, "ids")
	if err != nil {
		badQueryError(w, err)
		return
	}
	companyID, err := optionalIntQuery(r, "company_id")
	if err != nil {
		badQueryError(w, err)
		return
	}
	limit, err := intQueryDefault(r, "limit", 0)
	if err != nil {
		badQueryError(w, err)
		return
	}

	out, err := h.Service.SearchContacts(r.Context(), models.ContactSearch{Query: r.URL.Query().Get("q"), CompanyID: companyID, IDs: ids, Limit: limit})
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, out)
}

// idListQuery parses a comma-separated integer list; an absent parameter yields nil.
func idListQuery(r *http.Request, key string) ([]int, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return nil, nil
	}

	parts := strings.Split(raw, ",")
	ids := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		id, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %q is not an integer", key, p)
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (h *CWHandler) ListPriorities(w http.ResponseWriter, r *http.Request) {
	p, err := h.Service.ListPriorities(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, p)
}

func (h *CWHandler) GetBoard(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	b, err := h.Service.GetBoard(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrBoardNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, b)
}

func (h *CWHandler) ListBoardStatuses(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	st, err := h.Service.ListStatusesByBoard(r.Context(), id)
	if err != nil {
		internalServerError(w, err)
		return
	}

	if st == nil {
		st = []*models.TicketStatus{}
	}
	outputJSON(w, st)
}
