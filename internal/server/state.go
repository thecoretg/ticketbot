package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

const (
	initStatusKey    = "init_done"
	attemptNotifyKey = "attempt_notify"
)

type appState struct {
	InitDone      bool `json:"init_done"`
	AttemptNotify bool `json:"attempt_notify"`
}

type boolStateResult struct {
	isSet bool
	value bool
	err   error
}

func (s *Server) populateAppState(ctx context.Context) error {
	if err := s.setStateIfNotSet(ctx, initStatusKey); err != nil {
		return fmt.Errorf("checking init status: %w", err)
	}

	if err := s.setStateIfNotSet(ctx, attemptNotifyKey); err != nil {
		return fmt.Errorf("checking attempt notify value: %w", err)
	}

	return nil
}

func (s *Server) setAttemptNotify(ctx context.Context, attempt bool) error {
	return s.setBoolState(ctx, attemptNotifyKey, attempt)
}

func (s *Server) setStateIfNotSet(ctx context.Context, key string) error {
	r := s.getBoolState(ctx, key)
	if r.err != nil {
		slog.Warn("error getting app state", "key", key, "error", r.err)
	}

	if !r.isSet {
		slog.Debug("app state key is not set - setting to false", "key", key)
		return s.setBoolState(ctx, key, false)
	}

	slog.Debug("app state key is already set", "key", key, "value", r.value)
	return nil
}

func (s *Server) getBoolState(ctx context.Context, key string) boolStateResult {
	val, err := s.Queries.GetAppState(ctx, key)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return boolStateResult{
				isSet: false,
				value: false,
				err:   nil,
			}
		}
		return boolStateResult{
			isSet: false,
			value: false,
			err:   err,
		}
	}

	b := false
	if val == "true" {
		b = true
	}

	if val != "true" && val != "false" {
		err = fmt.Errorf("unexpected value found: %s", val)
	}

	return boolStateResult{
		isSet: true,
		value: b,
		err:   err,
	}
}

func (s *Server) setBoolState(ctx context.Context, key string, val bool) error {
	v := "false"
	if val {
		v = "true"
	}

	p := db.SetAppStateParams{
		Key:   key,
		Value: v,
	}

	if err := s.Queries.SetAppState(ctx, p); err != nil {
		return fmt.Errorf("setting key %s to %s: %w", key, v, err)
	}

	return nil
}
