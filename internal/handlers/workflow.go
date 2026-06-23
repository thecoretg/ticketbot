package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowHandler struct {
	Svc *workflow.Service
}

func NewWorkflowHandler(svc *workflow.Service) *WorkflowHandler {
	return &WorkflowHandler{Svc: svc}
}

// UpdateFields returns the catalog of updatable ticket fields for the panel's
// ticket_update op builder.
func (h *WorkflowHandler) UpdateFields(c *gin.Context) {
	outputJSON(c, h.Svc.UpdateFields())
}

func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	w, err := h.Svc.ListWorkflows(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, w)
}

func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	w, err := h.Svc.GetWorkflow(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrWorkflowNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, w)
}

func (h *WorkflowHandler) AddWorkflow(c *gin.Context) {
	p := &models.Workflow{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	w, err := h.Svc.AddWorkflow(c.Request.Context(), p)
	if err != nil {
		if errors.Is(err, workflow.ErrUnknownAction) || errors.Is(err, workflow.ErrInvalidConfig) {
			badPayloadError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, w)
}

func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	p := &models.Workflow{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}
	p.ID = id

	w, err := h.Svc.UpdateWorkflow(c.Request.Context(), p)
	if err != nil {
		if errors.Is(err, models.ErrWorkflowNotFound) {
			notFoundError(c, err)
			return
		}
		if errors.Is(err, workflow.ErrUnknownAction) || errors.Is(err, workflow.ErrInvalidConfig) {
			badPayloadError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, w)
}

func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Svc.DeleteWorkflow(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrWorkflowNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
