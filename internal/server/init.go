package server

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

func (s *Server) checkAndRunInit(ctx context.Context) error {
	done := s.getInit(ctx)

	if !done {
		if err := s.runInitialDeployment(ctx); err != nil {
			slog.Info("server initialization has not been run, running now")
			return fmt.Errorf("running initialization: %w", err)
		}

		if err := s.setInit(ctx, true); err != nil {
			slog.Warn("error setting init done to true", "error", err)
		}

		return nil
	}

	slog.Info("server initialization has already been done")
	return nil
}

func (s *Server) runInitialDeployment(ctx context.Context) error {
	key, err := s.BootstrapAdmin(ctx)
	if err != nil {
		return fmt.Errorf("bootstrapping initial admin key: %w", err)
	}

	if key != "" {
		slog.Info("bootstrap token created", "email", s.Config.InitialAdminEmail, "key", key)
		slog.Info("waiting 60 seconds - please copy the above key, as it will not be shown again")
		time.Sleep(60 * time.Minute)
	} else {
		slog.Info("bootstrap token already exists")
	}

	if err := s.InitAllHooks(); err != nil {
		return fmt.Errorf("initializing webhooks: %w", err)
	}

	return nil
}

func (s *Server) getInit(ctx context.Context) bool {
	r := s.getBoolState(ctx, initStatusKey)
	if r.err != nil {
		slog.Warn("error getting init status", "error", r.err)
	}

	return r.value
}

func (s *Server) setInit(ctx context.Context, done bool) error {
	return s.setBoolState(ctx, initStatusKey, done)
}
