package authsvc

import (
	"context"
	"errors"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

// fakeUsers satisfies repos.APIUserRepository via embedding; only GetForAuth is implemented.
type fakeUsers struct {
	repos.APIUserRepository
	byEmail map[string]*models.UserAuth
}

func (f *fakeUsers) GetForAuth(_ context.Context, email string) (*models.UserAuth, error) {
	if u, ok := f.byEmail[email]; ok {
		return u, nil
	}
	return nil, models.ErrAPIUserNotFound
}

// fakeSessions satisfies repos.SessionRepository via embedding; only Create is implemented.
type fakeSessions struct {
	repos.SessionRepository
}

func (f *fakeSessions) Create(_ context.Context, s *models.Session) (*models.Session, error) {
	return s, nil
}

func TestLoginHonoursPasswordLoginToggle(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("Passw0rd!"), bcrypt.MinCost)
	users := &fakeUsers{byEmail: map[string]*models.UserAuth{
		"admin@example.com": {ID: 1, EmailAddress: "admin@example.com", PasswordHash: hash},
		"bob@example.com":   {ID: 2, EmailAddress: "bob@example.com", PasswordHash: hash},
	}}
	cfg := &models.Config{PasswordLoginEnabled: false, SSOEnabled: true}
	s := New(users, &fakeSessions{}, nil, nil, cfg, "Admin@Example.com")

	if _, err := s.Login(context.Background(), "bob@example.com", "Passw0rd!"); !errors.Is(err, ErrPasswordLoginDisabled) {
		t.Fatalf("bob: got %v, want ErrPasswordLoginDisabled", err)
	}
	if res, err := s.Login(context.Background(), "admin@example.com", "Passw0rd!"); err != nil || res.Token == "" {
		t.Fatalf("break-glass admin: res=%+v err=%v", res, err)
	}

	cfg.PasswordLoginEnabled = true
	if _, err := s.Login(context.Background(), "bob@example.com", "Passw0rd!"); err != nil {
		t.Fatalf("bob with password login enabled: %v", err)
	}
}
