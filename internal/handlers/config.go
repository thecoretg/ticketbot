package handlers

import (
	"fmt"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/models"
)

type ConfigHandler struct {
	Service *config.Service
}

func NewConfigHandler(svc *config.Service) *ConfigHandler {
	return &ConfigHandler{Service: svc}
}

func (h *ConfigHandler) Get(w http.ResponseWriter, r *http.Request) {
	cfg, err := h.Service.Get(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, cfg)
}

func (h *ConfigHandler) Update(w http.ResponseWriter, r *http.Request) {
	p := &models.ConfigUpdateParams{}
	if err := decodeJSON(r, p); err != nil {
		badPayloadError(w, err)
		return
	}

	cfg, err := h.Service.Update(r.Context(), p)
	if err != nil {
		internalServerError(w, fmt.Errorf("updating config: %w", err))
		return
	}

	outputJSON(w, cfg)
}
