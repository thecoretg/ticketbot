package config

import (
	"context"
	"errors"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// fakeConfigRepo satisfies repos.ConfigRepository via embedding; Get and Upsert are implemented.
type fakeConfigRepo struct {
	repos.ConfigRepository
	stored models.Config
}

func (f *fakeConfigRepo) Get(context.Context) (*models.Config, error) {
	c := f.stored
	return &c, nil
}

func (f *fakeConfigRepo) Upsert(_ context.Context, c *models.Config) (*models.Config, error) {
	f.stored = *c
	out := *c
	return &out, nil
}

func ptr[T any](v T) *T { return &v }

func TestUpdateValidatesSSOToggles(t *testing.T) {
	base := models.DefaultConfig

	t.Run("sso needs env", func(t *testing.T) {
		repo := &fakeConfigRepo{stored: base}
		cfg := base
		s := New(repo, &cfg, nil, nil, false)
		_, err := s.Update(context.Background(), &models.ConfigUpdateParams{SSOEnabled: ptr(true)})
		if !errors.Is(err, ErrSSONotConfigured) {
			t.Fatalf("got %v, want ErrSSONotConfigured", err)
		}
		var ve ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("error %v is not a ValidationError", err)
		}
	})

	t.Run("password login needs sso", func(t *testing.T) {
		repo := &fakeConfigRepo{stored: base}
		cfg := base
		s := New(repo, &cfg, nil, nil, true)
		if _, err := s.Update(context.Background(), &models.ConfigUpdateParams{PasswordLoginEnabled: ptr(false)}); !errors.Is(err, ErrPasswordLoginNeedsSSO) {
			t.Fatalf("got %v, want ErrPasswordLoginNeedsSSO", err)
		}
		if _, err := s.Update(context.Background(), &models.ConfigUpdateParams{SSOEnabled: ptr(true), PasswordLoginEnabled: ptr(false)}); err != nil {
			t.Fatalf("both toggles together: %v", err)
		}
		if cfg.PasswordLoginEnabled || !cfg.SSOEnabled {
			t.Fatalf("shared config not updated: %+v", cfg)
		}
	})
}
