package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/user"
)

type UserHandler struct {
	Service *user.Service
}

func NewUserHandler(svc *user.Service) *UserHandler {
	return &UserHandler{Service: svc}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	u, err := h.Service.ListUsers(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, u)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	u, err := h.Service.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, u)
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	p := &models.APIUser{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	u, err := h.Service.InsertUser(c.Request.Context(), p.EmailAddress)
	if err != nil {
		if errors.Is(err, user.ErrUserAlreadyExists{Email: p.EmailAddress}) {
			conflictError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, u)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Service.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *UserHandler) ListAPIKeys(c *gin.Context) {
	k, err := h.Service.ListAPIKeys(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}

	outputJSON(c, k)
}

func (h *UserHandler) GetAPIKey(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	k, err := h.Service.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	outputJSON(c, k)
}

func (h *UserHandler) AddAPIKey(c *gin.Context) {
	p := &models.CreateAPIKeyPayload{}
	if err := c.ShouldBindJSON(p); err != nil {
		badPayloadError(c, err)
		return
	}

	k, err := h.Service.AddAPIKey(c.Request.Context(), p.Email)
	if err != nil {
		internalServerError(c, err)
		return
	}

	o := models.CreateAPIKeyResponse{
		Email: p.Email,
		Key:   k,
	}

	outputJSON(c, o)
}

func (h *UserHandler) DeleteAPIKey(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Service.DeleteAPIKey(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	c.Status(http.StatusOK)
}
