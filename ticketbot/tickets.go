package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"

	"github.com/gin-gonic/gin"
)

type cwData struct {
	ticket *connectwise.Ticket
	note   *connectwise.ServiceTicketNote
}

type storedData struct {
	ticket db.Ticket
	note   db.TicketNote
	board  db.Board
}

func (s *Server) addHooksGroup() {
	hooks := s.GinEngine.Group("/hooks")
	cw := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.Config.ExitOnError))
	cw.POST("/tickets", s.handleTickets)
}

func (s *Server) handleTickets(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling connectwise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("received no ticket ID or zero"))
		return
	}

	slog.Info("received payload from connectwise", "ticket_id", w.ID, "action", w.Action)
	switch w.Action {
	case "added", "updated":
		if err := s.processTicketPayload(c.Request.Context(), w.ID, w.Action, false, s.Config.AttemptNotify); err != nil {
			c.Error(fmt.Errorf("ticket %d: adding or updating the ticket into data storage: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)

	case "deleted":
		if err := s.softDeleteTicket(c.Request.Context(), w.ID); err != nil {
			c.Error(fmt.Errorf("ticket %d: deleting ticket and its notes: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *Server) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := s.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

// processTicketPayload serves as the primary handler for updating the data store with ticket data. It also will handle
// extra functionality such as ticket notifications.
func (s *Server) processTicketPayload(ctx context.Context, ticketID int, action string, overrideNotify, attemptNotify bool) error {
	// Lock the ticket so that extra calls don't interfere. Due to the nature of Connectwise updates will often
	// result in other hooks and actions taking place, which means a ticket rarely only sends one webhook payload.
	lock := s.getTicketLock(ticketID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	// Get the current data for the ticket via the Connectwise API.
	// This will be used to compare for changes with the store ticket.
	cwData, err := s.getCwData(ticketID)
	if err != nil {
		return fmt.Errorf("getting data from connectwise: %w", err)
	}

	storedData, err := s.getStoredData(ctx, cwData, overrideNotify)
	if err != nil {
		return fmt.Errorf("getting or creating stored data: %w", err)
	}

	// Insert or update the ticket into the store if it didn't exist or if there were changes.
	p := cwDataToUpdateTicketParams(cwData, storedData)
	storedData.ticket, err = s.Queries.UpdateTicket(ctx, p)
	if err != nil {
		return fmt.Errorf("updating ticket in store: %w", err)
	}

	notified := false
	if meetsMessageCriteria(action, storedData) && attemptNotify {
		if err := s.makeAndSendWebexMsgs(action, cwData, storedData); err != nil {
			return fmt.Errorf("processing webex messages: %w", err)
		}
		notified = true
	}

	// Always set notified to true if there is a note
	if storedData.note.ID != 0 {
		if err := s.setNotified(ctx, storedData.note.ID, true); err != nil {
			return fmt.Errorf("setting notified to true: %w", err)
		}
	}

	s.logTicketResult(action, storedData, notified)

	return nil
}

func (s *Server) softDeleteTicket(ctx context.Context, ticketID int) error {
	if err := s.Queries.SoftDeleteTicket(ctx, ticketID); err != nil {
		return fmt.Errorf("soft deleting ticket: %w", err)
	}
	slog.Debug("ticket soft deleted", "ticket_id", ticketID)

	return nil
}

func (s *Server) getCwData(ticketID int) (*cwData, error) {
	ticket, err := s.CWClient.GetTicket(ticketID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting ticket: %w", err)
	}

	note, err := s.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note: %w", err)
	}

	if note == nil {
		note = &connectwise.ServiceTicketNote{}
	}

	return &cwData{
		ticket: ticket,
		note:   note,
	}, nil
}

func (s *Server) getStoredData(ctx context.Context, cwData *cwData, overrideNotify bool) (*storedData, error) {
	// first check for or create board since it needs to exist before the ticket
	board, err := s.ensureBoardInStore(ctx, cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring board in store: %w", err)
	}

	// check for, or create ticket
	ticket, err := s.ensureTicketInStore(ctx, cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring ticket in store: %w", err)
	}

	// start with empty note, use existing or created note if there is a note in the ticket
	note := db.TicketNote{}
	if cwData.note.ID != 0 {
		note, err = s.ensureNoteInStore(ctx, cwData, overrideNotify)
		if err != nil {
			return nil, fmt.Errorf("ensuring note in store: %w", err)
		}
	}

	return &storedData{
		ticket: ticket,
		note:   note,
		board:  board,
	}, nil
}

func (s *Server) ensureTicketInStore(ctx context.Context, cwData *cwData) (db.Ticket, error) {
	ticket, err := s.Queries.GetTicket(ctx, cwData.ticket.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			owner := int32(cwData.ticket.Owner.ID)
			ticket, err = s.Queries.InsertTicket(ctx, db.InsertTicketParams{
				ID:           cwData.ticket.ID,
				Summary:      cwData.ticket.Summary,
				BoardID:      cwData.ticket.Board.ID,
				OwnerID:      &owner,
				Resources:    &cwData.ticket.Resources,
				UpdatedBy:    &cwData.ticket.Info.UpdatedBy,
				AddedToStore: time.Now(),
			})
			if err != nil {
				return db.Ticket{}, fmt.Errorf("inserting ticket into db: %w", err)
			}
			slog.Debug("inserted ticket into db", "ticket_id", ticket.ID, "summary", ticket.Summary)
			return ticket, nil
		} else {
			return db.Ticket{}, fmt.Errorf("getting ticket from storage: %w", err)
		}
	}

	slog.Debug("got existing ticket from store", "ticket_id", ticket.ID, "summary", ticket.Summary)
	return ticket, nil
}

func (s *Server) logTicketResult(action string, storedData *storedData, notified bool) {
	slog.Info("ticket processed", "ticket_id", storedData.ticket.ID, "action", action, "notified", notified)
}

// meetsMessageCriteria checks if a message would be allowed to send a notification,
// depending on if it was added or updated, if the note changed, and the board's notification settings.
func meetsMessageCriteria(action string, storedData *storedData) bool {
	if action == "added" {
		return storedData.board.NotifyEnabled
	}

	if action == "updated" {
		return !storedData.note.Notified && storedData.board.NotifyEnabled
	}

	return false
}

func cwDataToUpdateTicketParams(cwData *cwData, storedData *storedData) db.UpdateTicketParams {
	return db.UpdateTicketParams{
		ID:           cwData.ticket.ID,
		Summary:      cwData.ticket.Summary,
		BoardID:      cwData.ticket.Board.ID,
		OwnerID:      intToInt32Ptr(cwData.ticket.Owner.ID),
		Resources:    &cwData.ticket.Resources,
		UpdatedBy:    &cwData.ticket.Info.UpdatedBy,
		AddedToStore: storedData.ticket.AddedToStore,
	}
}
