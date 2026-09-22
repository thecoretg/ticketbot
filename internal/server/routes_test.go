package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/internal/service/sso"
	"github.com/thecoretg/ticketbot/models"
)

// TestNewHandlerRoutes registers every route (ServeMux panics on conflicting patterns) and checks
// that unauthenticated requests are rejected while the health check is open.
func TestNewHandlerRoutes(t *testing.T) {
	// NewNotifierHandler copies the service by value, so it needs a non-nil pointer.
	a := &App{Stores: &repos.AllRepos{}, Svc: &Services{Notifier: &notifier.Service{}, SSO: &sso.Service{}, OAuth: oauth.New(oauth.Params{Cfg: &models.Config{}})}}
	h := NewHandler(a, func() {})

	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/healthcheck", http.StatusOK},
		{http.MethodGet, "/users/me", http.StatusUnauthorized},
		{http.MethodGet, "/workflows/fields", http.StatusUnauthorized},
		{http.MethodGet, "/workflows/12", http.StatusUnauthorized},
		{http.MethodDelete, "/lists/1/items/2", http.StatusUnauthorized},
		{http.MethodPut, "/users/3/role", http.StatusUnauthorized},
		{http.MethodPut, "/config", http.StatusUnauthorized},
		{http.MethodGet, "/sso", http.StatusUnauthorized},
		{http.MethodDelete, "/sso/mappings/1", http.StatusUnauthorized},
		{http.MethodGet, "/auth/sso/start", http.StatusNotFound},
		{http.MethodGet, "/nope", http.StatusNotFound},
		{http.MethodPost, "/hooks/cw/tickets", http.StatusUnauthorized}, // unsigned callback
		{http.MethodGet, "/intake/stats", http.StatusUnauthorized},
		{http.MethodGet, "/workflows/export", http.StatusUnauthorized},
		{http.MethodGet, "/workflows/runs", http.StatusUnauthorized},
		{http.MethodGet, "/workflows/runs/abc", http.StatusUnauthorized},
		{http.MethodPost, "/workflows/import", http.StatusUnauthorized},
		{http.MethodGet, "/users/me/grants", http.StatusUnauthorized},
		{http.MethodDelete, "/users/grants/3/4", http.StatusUnauthorized},
		// MCP is off: its OAuth endpoints do not exist
		{http.MethodGet, "/.well-known/oauth-authorization-server", http.StatusNotFound},
		{http.MethodGet, "/.well-known/oauth-protected-resource/mcp", http.StatusNotFound},
		{http.MethodGet, "/oauth/authorize", http.StatusNotFound},
		{http.MethodPost, "/oauth/token", http.StatusNotFound},
		{http.MethodGet, "/oauth/consent", http.StatusNotFound},
		{http.MethodGet, "/", http.StatusOK},
		{http.MethodGet, "/app.js", http.StatusOK},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s: got %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}

	// dashboard assets must be revalidated on every load, or a CDN serves the previous build
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/app.js", nil))
	if cc := rec.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("GET /app.js Cache-Control = %q, want no-cache", cc)
	}

	// the old /panel/ address redirects to the root so bookmarks keep working
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/panel/", nil))
	if rec.Code/100 != 3 || rec.Header().Get("Location") != "/" {
		t.Errorf("GET /panel/: got %d -> %q, want redirect to /", rec.Code, rec.Header().Get("Location"))
	}
}

// TestNewHandlerMCPEnabled flips the switch on and checks the OAuth endpoints appear, with the
// consent path serving the dashboard.
func TestNewHandlerMCPEnabled(t *testing.T) {
	cfg := &models.Config{MCPEnabled: true}
	a := &App{Stores: &repos.AllRepos{}, Svc: &Services{Notifier: &notifier.Service{}, SSO: &sso.Service{}, OAuth: oauth.New(oauth.Params{Cfg: cfg, RootURL: "https://tb.example.com"})}}
	h := NewHandler(a, func() {})

	cases := []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/.well-known/oauth-authorization-server", http.StatusOK},
		{http.MethodGet, "/.well-known/oauth-protected-resource", http.StatusOK},
		{http.MethodGet, "/.well-known/oauth-protected-resource/mcp", http.StatusOK},
		{http.MethodGet, "/oauth/authorize", http.StatusBadRequest}, // no client_id
		{http.MethodPost, "/oauth/token", http.StatusUnauthorized},  // no client_id
		{http.MethodPost, "/oauth/register", http.StatusBadRequest},
		{http.MethodGet, "/oauth/authorize/info", http.StatusUnauthorized},
		{http.MethodPost, "/oauth/authorize/decide", http.StatusUnauthorized},
		{http.MethodGet, "/oauth/consent", http.StatusOK},
	}
	for _, c := range cases {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(c.method, c.path, nil))
		if rec.Code != c.want {
			t.Errorf("%s %s: got %d, want %d", c.method, c.path, rec.Code, c.want)
		}
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/oauth/consent", nil))
	if ct := rec.Header().Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Errorf("GET /oauth/consent Content-Type = %q, want text/html", ct)
	}
}
