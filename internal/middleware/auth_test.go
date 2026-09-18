package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/thecoretg/ticketbot/models"
)

func TestRequireRole(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })

	cases := []struct {
		name string
		user *models.APIUser
		min  models.Role
		want int
	}{
		{"unauthenticated", nil, models.RoleViewer, http.StatusUnauthorized},
		{"viewer meets viewer", &models.APIUser{Role: models.RoleViewer}, models.RoleViewer, http.StatusNoContent},
		{"viewer below editor", &models.APIUser{Role: models.RoleViewer}, models.RoleEditor, http.StatusForbidden},
		{"editor meets editor", &models.APIUser{Role: models.RoleEditor}, models.RoleEditor, http.StatusNoContent},
		{"editor below admin", &models.APIUser{Role: models.RoleEditor}, models.RoleAdmin, http.StatusForbidden},
		{"admin meets everything", &models.APIUser{Role: models.RoleAdmin}, models.RoleAdmin, http.StatusNoContent},
		{"unknown role is nothing", &models.APIUser{Role: "root"}, models.RoleViewer, http.StatusForbidden},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if c.user != nil {
				req = req.WithContext(withUser(context.Background(), c.user))
			}
			rec := httptest.NewRecorder()
			RequireRole(c.min)(ok).ServeHTTP(rec, req)
			if rec.Code != c.want {
				t.Fatalf("got %d, want %d", rec.Code, c.want)
			}
		})
	}
}

func TestUserIDWithoutUser(t *testing.T) {
	if got := UserID(context.Background()); got != 0 {
		t.Fatalf("UserID on empty context = %d, want 0", got)
	}
}
