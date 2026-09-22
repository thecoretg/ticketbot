package handlers

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"

	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/models"
)

// OAuthHandler serves the authorization server in front of the MCP endpoint and the
// Connected apps API. The consent screen itself is a dashboard view; Authorize forwards to it.
type OAuthHandler struct {
	svc *oauth.Service
}

func NewOAuthHandler(svc *oauth.Service) *OAuthHandler {
	return &OAuthHandler{svc: svc}
}

func (h *OAuthHandler) ProtectedResourceMetadata(w http.ResponseWriter, _ *http.Request) {
	outputJSON(w, h.svc.ProtectedResourceMetadata())
}

func (h *OAuthHandler) ServerMetadata(w http.ResponseWriter, _ *http.Request) {
	outputJSON(w, h.svc.ServerMetadata())
}

// writeOAuthError answers in the RFC 6749 error shape, or redirects when the error says so.
func writeOAuthError(w http.ResponseWriter, r *http.Request, err error) {
	var oe *oauth.Error
	if !errors.As(err, &oe) {
		internalServerError(w, err)
		return
	}
	if oe.RedirectTo != "" {
		http.Redirect(w, r, oe.RedirectTo, http.StatusFound)
		return
	}
	writeJSON(w, oe.Status, oe)
}

func (h *OAuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req oauth.RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, oauth.Error{Code: "invalid_client_metadata", Description: "body must be JSON client metadata"})
		return
	}
	res, err := h.svc.Register(r.Context(), req)
	if err != nil {
		writeOAuthError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

// Authorize validates the request and hands the browser to the dashboard's consent view with
// the same query string. The view signs the user in if needed, then calls ConsentInfo and Decide.
func (h *OAuthHandler) Authorize(w http.ResponseWriter, r *http.Request) {
	if _, err := h.svc.ParseAuthorize(r.Context(), r.URL.Query()); err != nil {
		writeOAuthError(w, r, err)
		return
	}
	http.Redirect(w, r, oauth.ConsentPath+"?"+r.URL.RawQuery, http.StatusFound)
}

// ConsentInfo re-validates the forwarded query and describes the request for the consent view.
func (h *OAuthHandler) ConsentInfo(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.ParseAuthorize(r.Context(), r.URL.Query())
	if err != nil {
		var oe *oauth.Error
		if errors.As(err, &oe) {
			writeJSON(w, http.StatusBadRequest, oe)
			return
		}
		internalServerError(w, err)
		return
	}
	outputJSON(w, h.svc.ConsentInfo(req))
}

type decideRequest struct {
	// Query is the consent view's location.search, with or without the leading "?".
	Query   string   `json:"query"`
	Approve bool     `json:"approve"`
	Scopes  []string `json:"scopes"`
}

// Decide records the user's answer and returns where the view should send the browser.
func (h *OAuthHandler) Decide(w http.ResponseWriter, r *http.Request) {
	var p decideRequest
	if err := decodeJSON(r, &p); err != nil {
		badPayloadError(w, err)
		return
	}
	if len(p.Query) > 0 && p.Query[0] == '?' {
		p.Query = p.Query[1:]
	}
	q, err := url.ParseQuery(p.Query)
	if err != nil {
		badPayloadError(w, err)
		return
	}
	req, err := h.svc.ParseAuthorize(r.Context(), q)
	if err != nil {
		var oe *oauth.Error
		if errors.As(err, &oe) {
			writeJSON(w, http.StatusBadRequest, oe)
			return
		}
		internalServerError(w, err)
		return
	}
	if !p.Approve {
		outputJSON(w, M{"redirect": h.svc.Deny(req)})
		return
	}
	target, err := h.svc.Approve(r.Context(), middleware.UserID(r.Context()), req, p.Scopes)
	if err != nil {
		var oe *oauth.Error
		if errors.As(err, &oe) {
			writeJSON(w, http.StatusBadRequest, oe)
			return
		}
		internalServerError(w, err)
		return
	}
	outputJSON(w, M{"redirect": target})
}

func (h *OAuthHandler) Token(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, oauth.Error{Code: "invalid_request", Description: "body must be application/x-www-form-urlencoded"})
		return
	}
	res, err := h.svc.Token(r.Context(), r.PostForm)
	if err != nil {
		writeOAuthError(w, r, err)
		return
	}
	outputJSON(w, res)
}

func (h *OAuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, oauth.Error{Code: "invalid_request", Description: "body must be application/x-www-form-urlencoded"})
		return
	}
	if err := h.svc.Revoke(r.Context(), r.PostForm.Get("token")); err != nil {
		internalServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// ListMyGrants and RevokeMyGrant back the Connected apps section of the profile page.
func (h *OAuthHandler) ListMyGrants(w http.ResponseWriter, r *http.Request) {
	h.listGrants(w, r, middleware.UserID(r.Context()))
}

func (h *OAuthHandler) RevokeMyGrant(w http.ResponseWriter, r *http.Request) {
	h.revokeGrant(w, r, middleware.UserID(r.Context()))
}

// ListUserGrants and RevokeUserGrant are the admin's view of another user's grants.
func (h *OAuthHandler) ListUserGrants(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	h.listGrants(w, r, id)
}

func (h *OAuthHandler) RevokeUserGrant(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}
	h.revokeGrant(w, r, id)
}

func (h *OAuthHandler) listGrants(w http.ResponseWriter, r *http.Request, userID int) {
	gs, err := h.svc.ListGrants(r.Context(), userID)
	if err != nil {
		internalServerError(w, err)
		return
	}
	outputJSON(w, gs)
}

func (h *OAuthHandler) revokeGrant(w http.ResponseWriter, r *http.Request, userID int) {
	grantID, err := strconv.Atoi(r.PathValue("grant_id"))
	if err != nil {
		errJSON(w, http.StatusBadRequest, errors.New("grant_id is not a valid integer"))
		return
	}
	if err := h.svc.RevokeGrant(r.Context(), userID, grantID); err != nil {
		if errors.Is(err, models.ErrOAuthGrantNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}
	w.WriteHeader(http.StatusOK)
}
