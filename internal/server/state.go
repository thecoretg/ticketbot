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

func (cl *Client) populateAppState(ctx context.Context) error {
	if err := cl.setStateIfNotSet(ctx, initStatusKey); err != nil {
		return fmt.Errorf("checking init status: %w", err)
	}

	if err := cl.setStateIfNotSet(ctx, attemptNotifyKey); err != nil {
		return fmt.Errorf("checking attempt notify value: %w", err)
	}

	return nil
}

func (cl *Client) setAttemptNotify(ctx context.Context, attempt bool) error {
	return cl.setBoolState(ctx, attemptNotifyKey, attempt)
}

func (cl *Client) setStateIfNotSet(ctx context.Context, key string) error {
	r := cl.getBoolState(ctx, key)
	if r.err != nil {
		slog.Warn("error getting app state", "key", key, "error", r.err)
	}

	if !r.isSet {
		slog.Debug("app state key is not set - setting to false", "key", key)
		return cl.setBoolState(ctx, key, false)
	}

	slog.Debug("app state key is already set", "key", key, "value", r.value)
	return nil
}

func (cl *Client) getBoolState(ctx context.Context, key string) boolStateResult {
	val, err := cl.Queries.GetAppState(ctx, key)
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

func (cl *Client) setBoolState(ctx context.Context, key string, val bool) error {
	v := "false"
	if val {
		v = "true"
	}

	p := db.SetAppStateParams{
		Key:   key,
		Value: v,
	}

	if err := cl.Queries.SetAppState(ctx, p); err != nil {
		return fmt.Errorf("setting key %s to %s: %w", key, v, err)
	}

	return nil
}
