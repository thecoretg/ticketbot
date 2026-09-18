package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/lists"
	"github.com/thecoretg/ticketbot/models"
)

type ListsHandler struct {
	Svc *lists.Service
}

func NewListsHandler(svc *lists.Service) *ListsHandler {
	return &ListsHandler{Svc: svc}
}

// Types handles GET /lists/types: the item types a list can hold.
func (h *ListsHandler) Types(c *gin.Context) {
	outputJSON(c, h.Svc.Types())
}

// List handles GET /lists.
func (h *ListsHandler) List(c *gin.Context) {
	ls, err := h.Svc.List(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}
	if ls == nil {
		ls = []*models.List{}
	}
	outputJSON(c, ls)
}

// Get handles GET /lists/:id: the list, its labelled items, and the rules that use it.
func (h *ListsHandler) Get(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	d, err := h.Svc.Get(c.Request.Context(), id)
	if err != nil {
		h.listError(c, err)
		return
	}
	outputJSON(c, d)
}

type listRequest struct {
	Name        string              `json:"name"`
	ItemType    models.ListItemType `json:"item_type,omitempty"`
	Description string              `json:"description"`
}

// Create handles POST /lists.
func (h *ListsHandler) Create(c *gin.Context) {
	req, ok := bindList(c)
	if !ok {
		return
	}

	l, err := h.Svc.Create(c.Request.Context(), &models.List{Name: req.Name, ItemType: req.ItemType, Description: req.Description})
	if err != nil {
		h.listError(c, err)
		return
	}
	c.JSON(http.StatusCreated, l)
}

// Update handles PUT /lists/:id: name and description only; the item type cannot change.
func (h *ListsHandler) Update(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}
	req, ok := bindList(c)
	if !ok {
		return
	}

	ctx := c.Request.Context()
	if req.ItemType != "" {
		current, err := h.Svc.Lists.Get(ctx, id)
		if err != nil {
			h.listError(c, err)
			return
		}
		if current.ItemType != req.ItemType {
			badPayloadError(c, errors.New("item_type cannot be changed"))
			return
		}
	}

	l, err := h.Svc.Update(ctx, id, req.Name, req.Description)
	if err != nil {
		h.listError(c, err)
		return
	}
	outputJSON(c, l)
}

// Delete handles DELETE /lists/:id. A list still referenced by a workflow rule returns 409 with
// the references.
func (h *ListsHandler) Delete(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Svc.Delete(c.Request.Context(), id); err != nil {
		h.listError(c, err)
		return
	}
	resultJSON(c, "list deleted")
}

type listItemRequest struct {
	ItemID int `json:"item_id"`
}

// AddItem handles POST /lists/:id/items.
func (h *ListsHandler) AddItem(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	var req listItemRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(c, err)
		return
	}
	if req.ItemID <= 0 {
		badPayloadError(c, errors.New("item_id is required"))
		return
	}

	it, err := h.Svc.AddItem(c.Request.Context(), id, req.ItemID)
	if err != nil {
		h.listError(c, err)
		return
	}
	c.JSON(http.StatusCreated, it)
}

// RemoveItem handles DELETE /lists/:id/items/:item_id.
func (h *ListsHandler) RemoveItem(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}
	itemID, err := strconv.Atoi(c.Param("item_id"))
	if err != nil {
		errJSON(c, http.StatusBadRequest, errors.New(c.Param("item_id")+" is not a valid integer"))
		return
	}

	if err := h.Svc.RemoveItem(c.Request.Context(), id, itemID); err != nil {
		h.listError(c, err)
		return
	}
	resultJSON(c, "item removed")
}

func bindList(c *gin.Context) (listRequest, bool) {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	var req listRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(c, err)
		return req, false
	}
	return req, true
}

func (h *ListsHandler) listError(c *gin.Context, err error) {
	var inUse *models.ListInUseError
	var invalid *models.ListValidationError
	switch {
	case errors.Is(err, models.ErrListNotFound), errors.Is(err, models.ErrListItemNotFound):
		notFoundError(c, err)
	case errors.Is(err, models.ErrListNameTaken):
		conflictError(c, err)
	case errors.As(err, &inUse):
		c.JSON(http.StatusConflict, gin.H{"error": err.Error(), "references": inUse.Refs})
	case errors.As(err, &invalid):
		errJSON(c, http.StatusBadRequest, err)
	case errors.Is(err, models.ErrContactNotFound), errors.Is(err, models.ErrCompanyNotFound):
		errJSON(c, http.StatusBadRequest, err)
	default:
		internalServerError(c, err)
	}
}
