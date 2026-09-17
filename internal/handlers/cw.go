package handlers

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// ListCompanies handles GET /cw/companies?q=&ids=1,2&limit=
func (h *CWHandler) ListCompanies(c *gin.Context) {
	ids, err := idListQuery(c, "ids")
	if err != nil {
		badQueryError(c, err)
		return
	}
	limit, err := intQueryDefault(c, "limit", 0)
	if err != nil {
		badQueryError(c, err)
		return
	}

	out, err := h.Service.SearchCompanies(c.Request.Context(), models.CompanySearch{Query: c.Query("q"), IDs: ids, Limit: limit})
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, out)
}

// ListContacts handles GET /cw/contacts?q=&company_id=&ids=1,2&limit=
func (h *CWHandler) ListContacts(c *gin.Context) {
	ids, err := idListQuery(c, "ids")
	if err != nil {
		badQueryError(c, err)
		return
	}
	companyID, err := optionalIntQuery(c, "company_id")
	if err != nil {
		badQueryError(c, err)
		return
	}
	limit, err := intQueryDefault(c, "limit", 0)
	if err != nil {
		badQueryError(c, err)
		return
	}

	out, err := h.Service.SearchContacts(c.Request.Context(), models.ContactSearch{Query: c.Query("q"), CompanyID: companyID, IDs: ids, Limit: limit})
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, out)
}

// idListQuery parses a comma-separated integer list; an absent parameter yields nil.
func idListQuery(c *gin.Context, key string) ([]int, error) {
	raw := strings.TrimSpace(c.Query(key))
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

func (h *CWHandler) ListPriorities(c *gin.Context) {
	p, err := h.Service.ListPriorities(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, p)
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

func (h *CWHandler) ListBoardStatuses(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	st, err := h.Service.ListStatusesByBoard(c.Request.Context(), id)
	if err != nil {
		internalServerError(c, err)
		return
	}

	if st == nil {
		st = []*models.TicketStatus{}
	}
	outputJSON(c, st)
}
