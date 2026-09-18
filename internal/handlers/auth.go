package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/service/authsvc"
	"github.com/thecoretg/ticketbot/internal/service/sso"
)

type AuthHandler struct {
	svc *authsvc.Service
	sso *sso.Service
}

func NewAuthHandler(svc *authsvc.Service, ssoSvc *sso.Service) *AuthHandler {
	return &AuthHandler{svc: svc, sso: ssoSvc}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *AuthHandler) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	result, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) || errors.Is(err, authsvc.ErrNoPassword) {
			writeJSON(w, http.StatusUnauthorized, M{"error": "invalid email or password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "login failed"})
		return
	}

	if result.TOTPRequired {
		writeJSON(w, http.StatusOK, M{"ok": true, "totp_required": true, "pending_token": result.PendingToken})
		return
	}

	setSessionCookie(w, result.Token, 24*time.Hour)
	writeJSON(w, http.StatusOK, M{"ok": true, "reset_required": result.ResetRequired, "totp_setup_required": result.TOTPSetupRequired})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) HandleChangePassword(w http.ResponseWriter, r *http.Request) {
	var req changePasswordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	if req.NewPassword == "" {
		writeJSON(w, http.StatusBadRequest, M{"error": "new password cannot be empty"})
		return
	}

	userID := middleware.UserID(r.Context())
	if err := h.svc.ChangePassword(r.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, M{"error": "current password is incorrect"})
			return
		}
		if errors.Is(err, authsvc.ErrWeakPassword) {
			writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "failed to change password"})
		return
	}

	writeJSON(w, http.StatusOK, M{"ok": true})
}

// HandleLogout ends whichever session the browser holds: the password session, the Microsoft
// session, or both.
func (h *AuthHandler) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if ck, err := r.Cookie(cookieName); err == nil && ck.Value != "" {
		_ = h.svc.Logout(r.Context(), ck.Value)
	}
	setSessionCookie(w, "", -time.Second)

	if ck, err := r.Cookie(sso.SessionCookie); err == nil && ck.Value != "" && h.sso != nil {
		if err := h.sso.EndSession(r.Context(), ck.Value); err != nil {
			slog.Warn("ending sso session", "error", err)
		}
		http.SetCookie(w, &http.Cookie{Name: sso.SessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}

	writeJSON(w, http.StatusOK, M{"ok": true})
}
