package handlers

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/models"
)

type UserHandler struct {
	Service *user.Service
}

func NewUserHandler(svc *user.Service) *UserHandler {
	return &UserHandler{Service: svc}
}

func (h *UserHandler) GetCurrentUser(c *gin.Context) {
	authenticatedUserID := c.GetInt("user_id")

	slog.Info("get current user called", "authenticated_user_id", authenticatedUserID)

	u, err := h.Service.GetUser(c.Request.Context(), authenticatedUserID)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	slog.Info("returning current user", "user_id", u.ID, "email", u.EmailAddress)
	outputJSON(c, u)
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

type createUserRequest struct {
	EmailAddress string `json:"email_address"`
	Password     string `json:"password"` // optional; if set, user must reset on first login
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var p createUserRequest
	if err := c.ShouldBindJSON(&p); err != nil {
		badPayloadError(c, err)
		return
	}

	var (
		u   *models.APIUser
		err error
	)

	if p.Password != "" {
		if err := authsvc.ValidatePassword(p.Password); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		u, err = h.Service.InsertUserWithPassword(c.Request.Context(), p.EmailAddress, p.Password)
	} else {
		u, err = h.Service.InsertUser(c.Request.Context(), p.EmailAddress)
	}

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

	authenticatedUserID := c.GetInt("user_id")

	slog.Info("user deletion requested",
		"authenticated_user_id", authenticatedUserID,
		"target_user_id", id)

	if err := h.Service.DeleteUser(c.Request.Context(), id, authenticatedUserID); err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(c, err)
			return
		}
		if errors.Is(err, user.ErrCannotDeleteSelf{}) {
			c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		internalServerError(c, err)
		return
	}

	slog.Info("user deleted successfully",
		"authenticated_user_id", authenticatedUserID,
		"deleted_user_id", id)

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
