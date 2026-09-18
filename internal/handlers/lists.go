package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

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
func (h *ListsHandler) Types(w http.ResponseWriter, r *http.Request) {
	outputJSON(w, h.Svc.Types())
}

// List handles GET /lists.
func (h *ListsHandler) List(w http.ResponseWriter, r *http.Request) {
	ls, err := h.Svc.List(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}
	if ls == nil {
		ls = []*models.List{}
	}
	outputJSON(w, ls)
}

// Get handles GET /lists/:id: the list, its labelled items, and the rules that use it.
func (h *ListsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	d, err := h.Svc.Get(r.Context(), id)
	if err != nil {
		h.listError(w, err)
		return
	}
	outputJSON(w, d)
}

type listRequest struct {
	Name        string              `json:"name"`
	ItemType    models.ListItemType `json:"item_type,omitempty"`
	Description string              `json:"description"`
}

// Create handles POST /lists.
func (h *ListsHandler) Create(w http.ResponseWriter, r *http.Request) {
	req, ok := bindList(w, r)
	if !ok {
		return
	}

	l, err := h.Svc.Create(r.Context(), &models.List{Name: req.Name, ItemType: req.ItemType, Description: req.Description})
	if err != nil {
		h.listError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, l)
}

// Update handles PUT /lists/:id: name and description only; the item type cannot change.
func (h *ListsHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	req, ok := bindList(w, r)
	if !ok {
		return
	}

	ctx := r.Context()
	if req.ItemType != "" {
		current, err := h.Svc.Lists.Get(ctx, id)
		if err != nil {
			h.listError(w, err)
			return
		}
		if current.ItemType != req.ItemType {
			badPayloadError(w, errors.New("item_type cannot be changed"))
			return
		}
	}

	l, err := h.Svc.Update(ctx, id, req.Name, req.Description)
	if err != nil {
		h.listError(w, err)
		return
	}
	outputJSON(w, l)
}

// Delete handles DELETE /lists/:id. A list still referenced by a workflow rule returns 409 with
// the references.
func (h *ListsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	if err := h.Svc.Delete(r.Context(), id); err != nil {
		h.listError(w, err)
		return
	}
	resultJSON(w, "list deleted")
}

type listItemRequest struct {
	ItemID int `json:"item_id"`
}

// AddItem handles POST /lists/:id/items.
func (h *ListsHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req listItemRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.ItemID <= 0 {
		badPayloadError(w, errors.New("item_id is required"))
		return
	}

	it, err := h.Svc.AddItem(r.Context(), id, req.ItemID)
	if err != nil {
		h.listError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, it)
}

// RemoveItem handles DELETE /lists/:id/items/:item_id.
func (h *ListsHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	itemID, err := strconv.Atoi(r.PathValue("item_id"))
	if err != nil {
		errJSON(w, http.StatusBadRequest, errors.New(r.PathValue("item_id")+" is not a valid integer"))
		return
	}

	if err := h.Svc.RemoveItem(r.Context(), id, itemID); err != nil {
		h.listError(w, err)
		return
	}
	resultJSON(w, "item removed")
}

func bindList(w http.ResponseWriter, r *http.Request) (listRequest, bool) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req listRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(w, err)
		return req, false
	}
	return req, true
}

func (h *ListsHandler) listError(w http.ResponseWriter, err error) {
	var inUse *models.ListInUseError
	var invalid *models.ListValidationError
	switch {
	case errors.Is(err, models.ErrListNotFound), errors.Is(err, models.ErrListItemNotFound):
		notFoundError(w, err)
	case errors.Is(err, models.ErrListNameTaken):
		conflictError(w, err)
	case errors.As(err, &inUse):
		writeJSON(w, http.StatusConflict, M{"error": err.Error(), "references": inUse.Refs})
	case errors.As(err, &invalid):
		errJSON(w, http.StatusBadRequest, err)
	case errors.Is(err, models.ErrContactNotFound), errors.Is(err, models.ErrCompanyNotFound):
		errJSON(w, http.StatusBadRequest, err)
	default:
		internalServerError(w, err)
	}
}
