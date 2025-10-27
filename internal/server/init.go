package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

const (
	initStatusKey = "init_done"
)

func (s *Server) checkAndRunInit(ctx context.Context) error {
	done, err := s.getInitStatus(ctx)
	if err != nil {
		return fmt.Errorf("getting init status: %w", err)
	}

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

func (s *Server) getInitStatus(ctx context.Context) (bool, error) {
	status, err := s.Queries.GetAppState(ctx, initStatusKey)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("checking initialization status: %w", err)
	}

	switch status {
	case "true":
		return true, nil
	case "false":
		return false, nil
	default:
		slog.Warn("got unexpected init status value", "value", status)
		return false, nil
	}
}

func (s *Server) setInit(ctx context.Context, done bool) error {
	val := "false"
	if done {
		val = "true"
	}

	p := db.SetAppStateParams{
		Key:   initStatusKey,
		Value: val,
	}

	if err := s.Queries.SetAppState(ctx, p); err != nil {
		return fmt.Errorf("setting init status to %s: %w", val, err)
	}

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
