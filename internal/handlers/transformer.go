package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/transformer"
	"github.com/thecoretg/ticketbot/models"
)

type TransformerHandler struct {
	Svc *transformer.Service
}

func NewTransformerHandler(svc *transformer.Service) *TransformerHandler {
	return &TransformerHandler{Svc: svc}
}

func (h *TransformerHandler) ListRules(c *gin.Context) {
	r, err := h.Svc.ListTransformerRules(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}

func (h *TransformerHandler) GetRule(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	r, err := h.Svc.GetTransformerRule(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrTransformerRuleNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}

func (h *TransformerHandler) AddRule(c *gin.Context) {
	p := &models.TransformerRule{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	r, err := h.Svc.AddTransformerRule(c.Request.Context(), p)
	if err != nil {
		if errors.Is(err, transformer.ErrUnknownAction) || errors.Is(err, transformer.ErrInvalidConfig) {
			badPayloadError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}

func (h *TransformerHandler) UpdateRule(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	p := &models.TransformerRule{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}
	p.ID = id

	r, err := h.Svc.UpdateTransformerRule(c.Request.Context(), p)
	if err != nil {
		if errors.Is(err, models.ErrTransformerRuleNotFound) {
			notFoundError(c, err)
			return
		}
		if errors.Is(err, transformer.ErrUnknownAction) || errors.Is(err, transformer.ErrInvalidConfig) {
			badPayloadError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, r)
}

func (h *TransformerHandler) DeleteRule(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Svc.DeleteTransformerRule(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrTransformerRuleNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
