package oldserver

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/external/webex"
)

type syncTicketsPayload struct {
	BoardIDs []int `json:"board_ids"`
}

func (cl *Client) handleSyncTickets(c *gin.Context) {
	if cl.State.SyncingTickets {
		c.JSON(http.StatusOK, gin.H{"result": "sync already in progress"})
		return
	}

	p := &syncTicketsPayload{}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json request"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "ticket sync started"})
	go func() {
		if err := cl.syncOpenTickets(context.Background(), p.BoardIDs); err != nil {
			slog.Error("syncing connectwise tickets", "error", err)
		}
	}()
}

func (cl *Client) handleSyncWebexRooms(c *gin.Context) {
	if cl.State.SyncingWebexRooms {
		slog.Info("webex room sync requested, but one is already in progress. returning 'sync already in progress' result")
		c.JSON(http.StatusOK, gin.H{"result": "sync already in progress"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": "webex room sync started"})
	go func() {
		if err := cl.syncWebexRooms(context.Background()); err != nil {
			slog.Error("syncing webex rooms", "error", err)
		}
	}()
}

func (cl *Client) handleSyncBoards(c *gin.Context) {
	if cl.State.SyncingBoards {
		slog.Info("boards sync requested, but one is already in progress. returning 'sync already in progress' result")
	}

	c.JSON(http.StatusOK, gin.H{"result": "connectwise board sync started"})
	go func() {
		if err := cl.syncBoards(context.Background()); err != nil {
			slog.Error("syncing connectwise boards", "error", err)
		}
	}()
}

// syncOpenTickets finds all open tickets in Connectwise and loads them into the DB if they don't
// already exist. It does not attempt to notify since that would result in tons of notifications
// for already existing tickets.
func (cl *Client) syncOpenTickets(ctx context.Context, boardIDS []int) error {
	cl.setSyncingTickets(true)
	defer func() {
		cl.setSyncingTickets(false)
	}()

	slog.Debug("beginning syncing tickets")

	con := "closedFlag = false"
	if len(boardIDS) > 0 {
		slog.Info("board ids provided for ticket sync", "ids", boardIDS)
		con += fmt.Sprintf(" AND %s", boardIDParam(boardIDS))
	}

	params := map[string]string{
		"pageSize":   "100",
		"conditions": con,
	}

	slog.Debug("loading existing open tickets")
	openTickets, err := cl.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from CW: %w", err)
	}
	slog.Debug("got open tickets from connectwise", "total_tickets", len(openTickets))
	sem := make(chan struct{}, cl.Config.MaxConcurrentSyncs)
	var wg sync.WaitGroup
	errCh := make(chan error, len(openTickets))

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket psa.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := cl.processTicket(ctx, ticket.ID, "sync", true); err != nil {
				slog.Error("ticket sync error", "id", ticket.ID, "error", err)
				errCh <- fmt.Errorf("error syncing ticket %d: %w", ticket.ID, err)
			} else {
				errCh <- nil
			}
		}(ticket)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("syncing ticket", "error", err)
		}
	}
	slog.Info("syncing tickets complete")
	return nil
}

func (cl *Client) syncWebexRooms(ctx context.Context) error {
	cl.setSyncingWebexRooms(true)
	defer func() {
		cl.setSyncingWebexRooms(false)
	}()

	slog.Debug("beginning sync of webex rooms")
	w, err := cl.MessageSender.ListRooms(nil)
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
				p := db.UpsertWebexRoomParams{
					Name: r.Title,
					Type: r.Type,
				}

				if _, err := qtx.UpsertWebexRoom(ctx, p); err != nil {
					return fmt.Errorf("updating room %s: %w", id, err)
				}
				rlog.Debug("room updated in db", "id", existing.ID)
			}
		} else {
			p := db.UpsertWebexRoomParams{
				WebexID: id,
				Name:    r.Title,
				Type:    r.Type,
			}

			n, err := qtx.UpsertWebexRoom(ctx, p)
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

	slog.Debug("webex room sync complete")
	return nil
}

func (cl *Client) syncBoards(ctx context.Context) error {
	cl.setSyncingBoards(true)
	defer func() {
		cl.setSyncingBoards(false)
	}()

	slog.Debug("beginning sync of connectwise boards")
	cwb, err := cl.CWClient.ListBoards(nil)
	if err != nil {
		return fmt.Errorf("listing connectwise boards: %w", err)
	}
	slog.Debug("got boards from connectwise", "total_boards", len(cwb))
	boards := make(map[int]psa.Board, len(cwb))
	for _, b := range cwb {
		boards[b.ID] = b
	}

	dbBoards, err := cl.Queries.ListBoards(ctx)
	if err != nil {
		return fmt.Errorf("listing boards in database: %w", err)
	}

	dbByID := make(map[int]db.CwBoard, len(dbBoards))
	for _, d := range dbBoards {
		dbByID[d.ID] = d
	}

	tx, err := cl.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("beginning transaction pool: %w", err)
	}
	qtx := cl.Queries.WithTx(tx)

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	for id, b := range boards {
		if existing, ok := dbByID[id]; ok {
			if boardChanged(existing, b) {
				slog.Debug("update needed for board", "id", id)
				p := db.UpsertBoardParams{
					ID:   b.ID,
					Name: b.Name,
				}

				if _, err := qtx.UpsertBoard(ctx, p); err != nil {
					return fmt.Errorf("updating board: %w", err)
				}
				slog.Debug("board updated in db", "id", b.ID, "name", b.Name)
			}
		} else {
			p := db.UpsertBoardParams{
				ID:   b.ID,
				Name: b.Name,
			}

			n, err := qtx.UpsertBoard(ctx, p)
			if err != nil {
				return fmt.Errorf("inserting board: %w", err)
			}
			slog.Debug("board inserted into db", "id", n.ID, "name", n.Name)
		}
	}

	for _, d := range dbBoards {
		if _, ok := boards[d.ID]; !ok {
			if err := qtx.DeleteBoard(ctx, d.ID); err != nil {
				return fmt.Errorf("deleting board from db: %w", err)
			}
			slog.Debug("board deleted from db", "id", d.ID, "name", d.Name)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing tx: %w", err)
	}
	slog.Debug("board room sync complete")
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

func boardChanged(existing db.CwBoard, cwBoard psa.Board) bool {
	return existing.Name != cwBoard.Name
}

func boardIDParam(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	param := ""
	for i, id := range ids {
		param += fmt.Sprintf("board/id = %d", id)
		if i < len(ids)-1 {
			param += " OR "
		}
	}

	return fmt.Sprintf("(%s)", param)
}
