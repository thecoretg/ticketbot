package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/models"
)

type NotifierHandler struct {
	Svc notifier.Service
}

func NewNotifierHandler(svc *notifier.Service) *NotifierHandler {
	return &NotifierHandler{
		Svc: *svc,
	}
}

func (h *NotifierHandler) ListForwards(c *gin.Context) {
	ctx := c.Request.Context()
	filter := c.Query("filter")

	var n []*models.NotifierForwardFull
	var err error

	switch filter {
	case "active":
		n, err = h.Svc.ListForwardsActive(ctx)
	case "inactive":
		n, err = h.Svc.ListForwardsInactive(ctx)
	case "not-expired":
		n, err = h.Svc.ListForwardsNotExpired(ctx)
	default:
		n, err = h.Svc.ListForwardsFull(ctx)
	}

	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, n)
}

func (h *NotifierHandler) GetForward(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	f, err := h.Svc.GetForward(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, f)
}

func (h *NotifierHandler) AddUserForward(c *gin.Context) {
	p := &models.NotifierForward{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	f, err := h.Svc.AddForward(c.Request.Context(), p)
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, f)
}

func (h *NotifierHandler) UpdateUserForward(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	p := &models.NotifierForward{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	f, err := h.Svc.UpdateForward(c.Request.Context(), id, p)
	if err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, f)
}

func (h *NotifierHandler) DeleteUserForward(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Svc.DeleteForward(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrUserForwardNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
