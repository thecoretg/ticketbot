package handlers

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/config"
)

type ConfigHandler struct {
	Service *config.Service
}

func NewConfigHandler(svc *config.Service) *ConfigHandler {
	return &ConfigHandler{Service: svc}
}

func (h *ConfigHandler) Get(c *gin.Context) {
	cfg, err := h.Service.Get(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, cfg)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	p := &models.Config{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	cfg, err := h.Service.Update(c.Request.Context(), p)
	if err != nil {
		internalServerError(c, fmt.Errorf("updating config: %w", err))
		return
	}

	outputJSON(c, cfg)
}
