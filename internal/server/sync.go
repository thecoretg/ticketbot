package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

func (cl *Client) handleSyncTickets(c *gin.Context) {
	if cl.State.SyncingTickets {
		c.JSON(http.StatusOK, gin.H{"result": "sync already in progress"})
		return
	}

	c.Status(http.StatusOK)
	go func() {
		if err := cl.syncOpenTickets(context.Background(), cl.Config.MaxConcurrentPreloads); err != nil {
			slog.Error("syncing connectwise tickets", "error", err)
		}
	}()
}

func (cl *Client) handleSyncWebexRooms(c *gin.Context) {
	if cl.State.SyncingWebexRooms {
		c.JSON(http.StatusOK, gin.H{"result": "sync already in progress"})
		return
	}

	c.Status(http.StatusOK)
	go func() {
		if err := cl.syncWebexRooms(context.Background()); err != nil {
			slog.Error("syncing webex rooms", "error", err)
		}
	}()
}

// syncOpenTickets finds all open tickets in Connectwise and loads them into the DB if they don't
// already exist. It does not attempt to notify since that would result in tons of notifications
// for already existing tickets.
func (cl *Client) syncOpenTickets(ctx context.Context, maxConcurrent int) error {
	if err := cl.setSyncingTickets(ctx, true); err != nil {
		slog.Warn("error setting syncing tickets value to true", "error", err)
	}

	defer func() {
		if err := cl.setSyncingTickets(ctx, false); err != nil {
			slog.Warn("error setting syncing tickets value to false")
		}
	}()

	slog.Debug("beginning preloading tickets")

	params := map[string]string{
		"pageSize":   "100",
		"conditions": "closedFlag = false",
	}

	slog.Debug("loading existing open tickets")
	openTickets, err := cl.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from CW: %w", err)
	}
	slog.Debug("got open tickets from connectwise", "total_tickets", len(openTickets))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	errCh := make(chan error, len(openTickets))

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket psa.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := cl.processTicket(ctx, ticket.ID, "preload", true); err != nil {
				errCh <- fmt.Errorf("error preloading ticket %d: %w", ticket.ID, err)
			} else {
				errCh <- nil
			}
		}(ticket)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("preloading ticket", "error", err)
			if cl.Config.ExitOnError {
				slog.Debug("exiting because EXIT_ON_ERROR is enabled")
				return err
			}
		}
	}
	return nil
}

func (cl *Client) syncWebexRooms(ctx context.Context) error {
	//TODO: THIS IS MESSY

	slog.Debug("beginning sync of webex rooms")
	w, err := cl.WebexClient.ListRooms(nil)
	if err != nil {
		return fmt.Errorf("listing webex rooms: %w", err)
	}
	slog.Debug("got rooms from webex", "total_rooms", len(w))
	webexRooms := make(map[string]webex.Room, len(w))
	for _, r := range w {
		webexRooms[r.Id] = r
	}

	dbRooms, err := cl.Queries.ListWebexRooms(ctx)
	if err != nil {
		return fmt.Errorf("listing rooms in database: %w", err)
	}

	dbByWebexID := make(map[string]db.WebexRoom, len(dbRooms))
	for _, d := range dbRooms {
		dbByWebexID[d.WebexID] = d
	}

	tx, err := cl.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction pool: %w", err)
	}
	qtx := cl.Queries.WithTx(tx)

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for id, r := range webexRooms {
		rlog := slog.With(roomLogger(r))
		if existing, ok := dbByWebexID[id]; ok {
			if roomChanged(existing, r) {
				rlog.Debug("updates needed for room")
				p := db.UpdateWebexRoomParams{
					ID:   existing.ID,
					Name: r.Title,
					Type: r.Type,
				}

				if _, err := qtx.UpdateWebexRoom(ctx, p); err != nil {
					return fmt.Errorf("updating room %s: %w", id, err)
				}
				rlog.Debug("room updated in db", "id", existing.ID)
			}
		} else {
			p := db.InsertWebexRoomParams{
				WebexID: id,
				Name:    r.Title,
				Type:    r.Type,
			}

			n, err := qtx.InsertWebexRoom(ctx, p)
			if err != nil {
				return fmt.Errorf("inserting room %s: %w", id, err)
			}
			rlog.Debug("room inserted into db", "id", n.ID)
		}
	}

	for _, d := range dbRooms {
		if _, ok := webexRooms[d.WebexID]; !ok {
			if err := qtx.DeleteWebexRoom(ctx, d.ID); err != nil {
				return fmt.Errorf("deleteting room %d: %w", d.ID, err)
			}
			slog.Debug("room deleted from db", "id", d.ID, "webex_id", d.WebexID)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}

	return nil
}

func roomLogger(r webex.Room) slog.Attr {
	return slog.Group(
		"webex_room",
		slog.String("id", r.Id),
		slog.String("name", r.Title),
		slog.String("type", r.Type),
	)
}

func roomChanged(existing db.WebexRoom, webexRoom webex.Room) bool {
	return existing.Name != webexRoom.Title || existing.Type != webexRoom.Type
}
