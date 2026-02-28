package handlers

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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
func (h *TOTPHandler) HandleVerify(c *gin.Context) {
	var req totpVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	token, resetRequired, err := h.svc.VerifyTOTP(c.Request.Context(), req.PendingToken, req.Code)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) || errors.Is(err, authsvc.ErrInvalidTOTPCode) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired code"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "verification failed"})
		return
	}

	c.SetCookie(cookieName, token, int(24*time.Hour/time.Second), "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{"ok": true, "reset_required": resetRequired})
}

// HandleBeginSetup generates a new TOTP secret and QR code for the user.
// The secret is NOT stored yet — the user must confirm a valid code.
func (h *TOTPHandler) HandleBeginSetup(c *gin.Context) {
	userID := c.GetInt("user_id")
	secret, otpauthURL, qrPNG, err := h.svc.BeginSetup(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to begin 2FA setup"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
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
func (h *TOTPHandler) HandleConfirmSetup(c *gin.Context) {
	var req totpConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.GetInt("user_id")
	codes, err := h.svc.ConfirmSetup(c.Request.Context(), userID, req.Password, req.Code, req.Secret)
	if err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
			return
		}
		if errors.Is(err, authsvc.ErrInvalidTOTPCode) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid code — check your authenticator app"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to enable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true, "recovery_codes": codes})
}

type totpDisableRequest struct {
	Password string `json:"password"`
}

// HandleStatus returns whether the authenticated user has TOTP enabled.
func (h *TOTPHandler) HandleStatus(c *gin.Context) {
	userID := c.GetInt("user_id")
	enabled, err := h.svc.TOTPStatus(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get 2FA status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"enabled": enabled})
}

// HandleDisable removes TOTP from the account after password confirmation.
func (h *TOTPHandler) HandleDisable(c *gin.Context) {
	var req totpDisableRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	userID := c.GetInt("user_id")
	if err := h.svc.DisableTOTP(c.Request.Context(), userID, req.Password); err != nil {
		if errors.Is(err, authsvc.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "incorrect password"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to disable 2FA"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
