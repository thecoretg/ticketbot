package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/thecoretg/ticketbot/internal/service/transfer"
	"github.com/thecoretg/ticketbot/models"
)

// TransferHandler exports workflows as a bundle and imports one.
type TransferHandler struct {
	Svc *transfer.Service
}

func NewTransferHandler(svc *transfer.Service) *TransferHandler {
	return &TransferHandler{Svc: svc}
}

// Export handles GET /workflows/export?ids=1,2 (all workflows when ids is absent). The response
// is served as a download named for the day.
func (h *TransferHandler) Export(w http.ResponseWriter, r *http.Request) {
	var ids []int
	if raw := strings.TrimSpace(r.URL.Query().Get("ids")); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			id, err := strconv.Atoi(strings.TrimSpace(part))
			if err != nil {
				badQueryError(w, fmt.Errorf("ids must be integers: %w", err))
				return
			}
			ids = append(ids, id)
		}
	}
	b, err := h.Svc.Export(r.Context(), ids)
	if err != nil {
		internalServerError(w, err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="ticketbot-workflows-%s.json"`, b.ExportedAt.Format("2006-01-02")))
	outputJSON(w, b)
}

type importRequest struct {
	Bundle  *models.WorkflowBundle `json:"bundle"`
	Replace bool                   `json:"replace"`
}

// Import handles POST /workflows/import with {"bundle": <export file>, "replace": bool}.
func (h *TransferHandler) Import(w http.ResponseWriter, r *http.Request) {
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.Bundle == nil {
		badPayloadError(w, fmt.Errorf("bundle is required"))
		return
	}
	rep, err := h.Svc.Import(r.Context(), req.Bundle, models.ImportOptions{Replace: req.Replace})
	if err != nil {
		errJSON(w, http.StatusBadRequest, err)
		return
	}
	outputJSON(w, rep)
}
