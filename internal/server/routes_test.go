package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/sso"
)

// TestNewHandlerRoutes registers every route (ServeMux panics on conflicting patterns) and checks
// that unauthenticated requests are rejected while the health check is open.
func TestNewHandlerRoutes(t *testing.T) {
	// NewNotifierHandler copies the service by value, so it needs a non-nil pointer.
	a := &App{Env: &env.Env{HookSignatureEnforced: true}, Stores: &repos.AllRepos{}, Svc: &Services{Notifier: &notifier.Service{}, SSO: &sso.Service{}}}
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
