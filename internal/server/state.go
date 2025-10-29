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

type appState struct {
	SyncingTickets    bool `json:"syncing_tickets"`
	SyncingWebexRooms bool `json:"syncing_webex_rooms"`
}

var defaultAppState = &appState{
	SyncingTickets:    false,
	SyncingWebexRooms: false,
}

func (cl *Client) handleGetState(c *gin.Context) {
	as, err := cl.getAppState(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, as)
}

func (cl *Client) getAppState(ctx context.Context) (*appState, error) {
	ds, err := cl.Queries.GetAppState(ctx)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("no app state found, creating default")
			ds, err = cl.Queries.UpsertAppState(ctx, db.UpsertAppStateParams{})
			if err != nil {
				return nil, fmt.Errorf("creating default app state: %w", err)
			}
			return dbStateToAppState(ds), nil
		}
		return nil, fmt.Errorf("getting app state from db: %w", err)
	}

	return dbStateToAppState(ds), nil
}

func (cl *Client) setSyncingTickets(ctx context.Context, syncing bool) error {
	cl.State.SyncingTickets = syncing
	if err := cl.updateAppState(ctx, cl.State); err != nil {
		return fmt.Errorf("state was set in memory, but an error occured updating the db: %w", err)
	}

	return nil
}

func (cl *Client) setSyncingWebexRooms(ctx context.Context, syncing bool) error {
	cl.State.SyncingWebexRooms = syncing
	if err := cl.updateAppState(ctx, cl.State); err != nil {
		return fmt.Errorf("state was set in memory, but an error occured updating the db: %w", err)
	}

	return nil
}

func (cl *Client) updateAppState(ctx context.Context, as *appState) error {
	p := stateToParams(as)

	ds, err := cl.Queries.UpsertAppState(ctx, p)
	if err != nil {
		return fmt.Errorf("updating in db: %w", err)
	}

	cl.State = dbStateToAppState(ds)
	return nil
}

func stateToParams(as *appState) db.UpsertAppStateParams {
	return db.UpsertAppStateParams{
		SyncingTickets:    as.SyncingTickets,
		SyncingWebexRooms: as.SyncingWebexRooms,
	}
}

func dbStateToAppState(ds db.AppState) *appState {
	return &appState{
		SyncingTickets:    ds.SyncingTickets,
		SyncingWebexRooms: ds.SyncingWebexRooms,
	}
}
