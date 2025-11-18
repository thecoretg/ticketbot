package handler

import (
	"errors"
	"fmt"
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
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, badIntErrorOutput(c.Param("id")))
		return
	}

	u, err := h.Service.GetUser(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			c.JSON(http.StatusNotFound, errorOutput(err))
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, u)
}

func (h *UserHandler) DeleteUser(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, badIntErrorOutput(c.Param("id")))
		return
	}

	if err := h.Service.DeleteUser(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			c.JSON(http.StatusNotFound, errorOutput(err))
			return
		}
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}

func (h *UserHandler) ListAPIKeys(c *gin.Context) {
	k, err := h.Service.ListAPIKeys(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, k)
}

func (h *UserHandler) GetAPIKey(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, badIntErrorOutput(c.Param("id")))
		return
	}

	k, err := h.Service.GetAPIKey(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			c.JSON(http.StatusNotFound, errorOutput(err))
			return
		}
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, k)
}

func (h *UserHandler) AddAPIKey(c *gin.Context) {
	p := &struct {
		Email string `json:"email"`
	}{}

	if err := c.ShouldBindJSON(p); err != nil {
		c.Error(fmt.Errorf("bad json payload: %w", err))
		return
	}

	k, err := h.Service.AddAPIKey(c.Request.Context(), p.Email)
	if err != nil {
		c.Error(fmt.Errorf("adding api key: %w", err))
		return
	}

	c.JSON(http.StatusOK, gin.H{"key": k})
}

func (h *UserHandler) DeleteAPIKey(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, badIntErrorOutput(c.Param("id")))
		return
	}

	if err := h.Service.DeleteAPIKey(c.Request.Context(), id); err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			c.JSON(http.StatusNotFound, errorOutput(err))
			return
		}
		c.Error(err)
		return
	}

	c.Status(http.StatusOK)
}
