package handlers

import (
	"encoding/base64"
	"errors"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"net/http"
	"time"

	"github.com/thecoretg/ticketbot/internal/service/authsvc"
)

type TOTPHandler struct {
	svc *authsvc.Service
}

func NewTOTPHandler(svc *authsvc.Service) *TOTPHandler {
	return &TOTPHandler{svc: svc}
}

type totpVerifyRequest struct {
	PendingToken string `json:"pending_token"`
	Code         string `json:"code"`
}

// HandleVerify is called after password login when TOTP is enabled.
// The pending_token from the login response is exchanged for a real session.
func (h *TOTPHandler) HandleVerify(w http.ResponseWriter, r *http.Request) {
	var req totpVerifyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	token, resetRequired, recoveryCodeUsed, err := h.svc.VerifyTOTP(r.Context(), req.PendingToken, req.Code)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) || errors.Is(err, authsvc.ErrInvalidTOTPCode) {
			writeJSON(w, http.StatusUnauthorized, M{"error": "invalid or expired code"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "verification failed"})
		return
	}

	setSessionCookie(w, token, 24*time.Hour)
	writeJSON(w, http.StatusOK, M{"ok": true, "reset_required": resetRequired, "recovery_code_used": recoveryCodeUsed})
}

// HandleBeginSetup generates a new TOTP secret and QR code for the user.
// The secret is NOT stored yet — the user must confirm a valid code.
func (h *TOTPHandler) HandleBeginSetup(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	secret, otpauthURL, qrPNG, err := h.svc.BeginSetup(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "failed to begin 2FA setup"})
		return
	}

	writeJSON(w, http.StatusOK, M{
		"secret":      secret,
		"otpauth_url": otpauthURL,
		"qr_png":      base64.StdEncoding.EncodeToString(qrPNG),
	})
}

type totpConfirmRequest struct {
	Password string `json:"password"`
	Code     string `json:"code"`
	Secret   string `json:"secret"`
}

// HandleConfirmSetup validates the TOTP code against the provided secret,
// enables TOTP, and returns one-time recovery codes.
func (h *TOTPHandler) HandleConfirmSetup(w http.ResponseWriter, r *http.Request) {
	var req totpConfirmRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	userID := middleware.UserID(r.Context())
	codes, err := h.svc.ConfirmSetup(r.Context(), userID, req.Password, req.Code, req.Secret)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, M{"error": "incorrect password"})
			return
		}
		if errors.Is(err, authsvc.ErrInvalidTOTPCode) {
			writeJSON(w, http.StatusBadRequest, M{"error": "invalid code — check your authenticator app"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "failed to enable 2FA"})
		return
	}

	writeJSON(w, http.StatusOK, M{"ok": true, "recovery_codes": codes})
}

type totpDisableRequest struct {
	Password string `json:"password"`
}

// HandleStatus returns whether the authenticated user has TOTP enabled.
func (h *TOTPHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	enabled, err := h.svc.TOTPStatus(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, M{"error": "failed to get 2FA status"})
		return
	}
	writeJSON(w, http.StatusOK, M{"enabled": enabled})
}

// HandleDisable removes TOTP from the account after password confirmation.
func (h *TOTPHandler) HandleDisable(w http.ResponseWriter, r *http.Request) {
	var req totpDisableRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": "invalid request body"})
		return
	}

	userID := middleware.UserID(r.Context())
	if err := h.svc.DisableTOTP(r.Context(), userID, req.Password); err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			writeJSON(w, http.StatusUnauthorized, M{"error": "incorrect password"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, M{"error": "failed to disable 2FA"})
		return
	}

	writeJSON(w, http.StatusOK, M{"ok": true})
}
