package handlers

import (
	"errors"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"log/slog"
	"net/http"

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

func (h *UserHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	authenticatedUserID := middleware.UserID(r.Context())

	slog.Info("get current user called", "authenticated_user_id", authenticatedUserID)

	u, err := h.Service.GetUser(r.Context(), authenticatedUserID)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	slog.Info("returning current user", "user_id", u.ID, "email", u.EmailAddress)
	outputJSON(w, u)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	u, err := h.Service.ListUsers(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, u)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	u, err := h.Service.GetUser(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, u)
}

type createUserRequest struct {
	EmailAddress string      `json:"email_address"`
	Password     string      `json:"password"` // optional; if set, user must reset on first login
	Role         models.Role `json:"role"`     // optional; defaults to viewer
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var p createUserRequest
	if err := decodeJSON(r, &p); err != nil {
		badPayloadError(w, err)
		return
	}

	if p.Role == "" {
		p.Role = models.RoleViewer
	}

	var (
		u   *models.APIUser
		err error
	)

	if p.Password != "" {
		if err := authsvc.ValidatePassword(p.Password); err != nil {
			writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
			return
		}
		u, err = h.Service.InsertUserWithPassword(r.Context(), p.EmailAddress, p.Password, p.Role)
	} else {
		u, err = h.Service.InsertUser(r.Context(), p.EmailAddress, p.Role)
	}

	if err != nil {
		if errors.Is(err, user.ErrUserAlreadyExists{Email: p.EmailAddress}) {
			conflictError(w, err)
			return
		}
		if errors.Is(err, models.ErrInvalidRole) {
			writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, u)
}

type setRoleRequest struct {
	Role models.Role `json:"role"`
}

func (h *UserHandler) SetRole(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	var p setRoleRequest
	if err := decodeJSON(r, &p); err != nil {
		badPayloadError(w, err)
		return
	}

	u, err := h.Service.SetRole(r.Context(), id, p.Role, middleware.UserID(r.Context()))
	if err != nil {
		switch {
		case errors.Is(err, models.ErrAPIUserNotFound):
			notFoundError(w, err)
		case errors.Is(err, models.ErrInvalidRole):
			writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
		case errors.Is(err, user.ErrCannotChangeOwnRole{}), errors.Is(err, user.ErrRoleManagedByEntra{}):
			writeJSON(w, http.StatusForbidden, M{"error": err.Error()})
		default:
			internalServerError(w, err)
		}
		return
	}

	outputJSON(w, u)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	authenticatedUserID := middleware.UserID(r.Context())

	slog.Info("user deletion requested",
		"authenticated_user_id", authenticatedUserID,
		"target_user_id", id)

	if err := h.Service.DeleteUser(r.Context(), id, authenticatedUserID); err != nil {
		if errors.Is(err, models.ErrAPIUserNotFound) {
			notFoundError(w, err)
			return
		}
		if errors.Is(err, user.ErrCannotDeleteSelf{}) {
			writeJSON(w, http.StatusForbidden, M{"error": err.Error()})
			return
		}
		internalServerError(w, err)
		return
	}

	slog.Info("user deleted successfully",
		"authenticated_user_id", authenticatedUserID,
		"deleted_user_id", id)

	w.WriteHeader(http.StatusOK)
}

func (h *UserHandler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	k, err := h.Service.ListAPIKeys(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}

	outputJSON(w, k)
}

func (h *UserHandler) GetAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	k, err := h.Service.GetAPIKey(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	outputJSON(w, k)
}

func (h *UserHandler) AddAPIKey(w http.ResponseWriter, r *http.Request) {
	p := &models.CreateAPIKeyPayload{}
	if err := decodeJSON(r, p); err != nil {
		badPayloadError(w, err)
		return
	}

	k, err := h.Service.AddAPIKey(r.Context(), p.Email)
	if err != nil {
		internalServerError(w, err)
		return
	}

	o := models.CreateAPIKeyResponse{
		Email: p.Email,
		Key:   k,
	}

	outputJSON(w, o)
}

func (h *UserHandler) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	if err := h.Service.DeleteAPIKey(r.Context(), id); err != nil {
		if errors.Is(err, models.ErrAPIKeyNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
