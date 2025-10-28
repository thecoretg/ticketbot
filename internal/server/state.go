package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

const (
	debugKey             = "debug"
	attemptNotifyKey     = "attempt_notify"
	syncingTicketsKey    = "syncing_tickets"
	syncingWebexRoomsKey = "syncing_webex_rooms"
)

type appState struct {
	Debug             bool `json:"debug"`
	AttemptNotify     bool `json:"attempt_notify"`
	SyncingTickets    bool `json:"syncing_tickets"`
	SyncingWebexRooms bool `json:"syncing_webex_rooms"`
}

type boolStateResult struct {
	isSet bool
	value bool
	err   error
}

func (cl *Client) handleGetState(c *gin.Context) {
	if cl.State == nil {
		c.Error(errors.New("app state is nil"))
		return
	}

	c.JSON(http.StatusOK, cl.State)
}

func (cl *Client) populateAppState(ctx context.Context) error {
	if err := cl.setStateIfNotSet(ctx, debugKey, false); err != nil {
		return fmt.Errorf("checking debug value: %w", err)
	}

	if err := cl.setStateIfNotSet(ctx, attemptNotifyKey, false); err != nil {
		return fmt.Errorf("checking attempt notify value: %w", err)
	}

	if err := cl.setStateIfNotSet(ctx, syncingTicketsKey, false); err != nil {
		return fmt.Errorf("checking syncing tickets value: %w", err)
	}

	if err := cl.setStateIfNotSet(ctx, syncingWebexRoomsKey, false); err != nil {
		return fmt.Errorf("checking syncing webex rooms value: %w", err)
	}

	return nil
}

func (cl *Client) setDebug(ctx context.Context, debug bool) error {
	setLogLevel(debug)
	return cl.setBoolState(ctx, debugKey, debug)
}

func (cl *Client) setAttemptNotify(ctx context.Context, attempt bool) error {
	return cl.setBoolState(ctx, attemptNotifyKey, attempt)
}

func (cl *Client) setSyncingTickets(ctx context.Context, syncing bool) error {
	return cl.setBoolState(ctx, syncingTicketsKey, syncing)
}

func (cl *Client) setSyncingWebexRooms(ctx context.Context, syncing bool) error {
	return cl.setBoolState(ctx, syncingWebexRoomsKey, syncing)
}

func (cl *Client) setStateIfNotSet(ctx context.Context, key string, defaultState bool) error {
	r := cl.getBoolState(ctx, key)
	if r.err != nil {
		slog.Warn("error getting app state", "key", key, "error", r.err)
	}

	if !r.isSet {
		slog.Debug("app state key is not set - setting to false", "key", key)
		return cl.setBoolState(ctx, key, defaultState)
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
