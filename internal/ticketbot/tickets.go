package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"tctg-automation/pkg/connectwise"
	"time"

	"github.com/gin-gonic/gin"
)

type cwData struct {
	ticket *connectwise.Ticket
	note   *connectwise.ServiceTicketNote
}

type storedData struct {
	ticket *Ticket
	note   *TicketNote
	board  *Board
}

func (s *Server) addHooksGroup() {
	hooks := s.ginEngine.Group("/hooks")
	cw := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.config.ExitOnError))
	cw.POST("/tickets", s.handleTickets)
}

func (s *Server) handleTickets(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling connectwise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("ticket ID cannot be 0"))
		return
	}

	slog.Info("received ticket webhook", "id", w.ID, "action", w.Action)
	if w.Action == "added" || w.Action == "updated" {
		if err := s.addOrUpdateTicket(c.Request.Context(), w.ID, w.Action, false); err != nil {
			c.Error(fmt.Errorf("adding or updating the ticket into data storage: %w", err))
			return
		}

		c.Status(http.StatusNoContent)
	} else {
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := s.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

// addOrUpdateTicket serves as the primary handler for updating the data store with ticket data. It also will handle
// extra functionality such as ticket notifications.
func (s *Server) addOrUpdateTicket(ctx context.Context, ticketID int, action string, assumeNotify bool) error {
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

	storedData, err := s.getStoredData(cwData, assumeNotify)
	if err != nil {
		return fmt.Errorf("getting or creating stored data: %w", err)
	}

	storedData.ticket = cwTicketToStoreTicket(cwData)
	// Insert or update the ticket into the store if it didn't exist or if there were changes.
	if err := s.dataStore.UpsertTicket(storedData.ticket); err != nil {
		return fmt.Errorf("upserting ticket to store: %w", err)
	}

	// Use the action from the CW hook, whether the note is considered new, and if the board
	// has notifications enabled to determine what type of notification will be sent, if any.
	if meetsMessageCriteria(action, storedData) {
		if err := s.makeAndSendWebexMsgs(ctx, action, cwData, storedData); err != nil {
			return fmt.Errorf("processing webex messages: %w", err)
		}
		storedData.note.Notified = true
	}

	// Log the result
	if s.config.Debug {
		slog.Debug("ticket processed", "ticket_id", storedData.ticket.ID, "action", action, "latest_note_id", storedData.note.ID, "assume_notify", assumeNotify,
			"notified", storedData.note.Notified, "board_notify_enbabled", storedData.board.NotifyEnabled, "meets_message_criteria", meetsMessageCriteria(action, storedData),
		)
	} else {
		slog.Info("ticket processed", "ticket_id", storedData.ticket.ID, "action", action, "notified", storedData.note.Notified)
	}

	// Always set notified to true if there is a note
	if storedData.note.ID != 0 {
		if err := s.setNotified(storedData.note, true); err != nil {
			return fmt.Errorf("setting notified to true: %w", err)
		}
	}

	return nil
}

func (s *Server) getCwData(ticketID int) (*cwData, error) {
	ticket, err := s.cwClient.GetTicket(ticketID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting ticket: %w", err)
	}

	note, err := s.cwClient.GetMostRecentTicketNote(ticketID)
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

func (s *Server) getStoredData(cwData *cwData, assumeNotified bool) (*storedData, error) {
	// Get existing ticket from store - will be nil if it doesn't already exist.
	ticket, err := s.ensureTicketInStore(cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring ticket in store: %w", err)
	}

	note := &TicketNote{}
	if cwData.note.ID != 0 {
		note, err = s.ensureNoteInStore(cwData, assumeNotified)
		if err != nil {
			return nil, fmt.Errorf("ensuring note in store: %w", err)
		}
	}

	board, err := s.ensureBoardInStore(cwData)
	if err != nil {
		return nil, fmt.Errorf("ensuring board in store: %w", err)
	}

	return &storedData{
		ticket: ticket,
		note:   note,
		board:  board,
	}, nil
}

func (s *Server) ensureTicketInStore(cwData *cwData) (*Ticket, error) {
	ticket, err := s.dataStore.GetTicket(cwData.ticket.ID)
	if err != nil {
		return nil, fmt.Errorf("getting ticket from storage: %w", err)
	}

	if ticket == nil {
		ticket = cwTicketToStoreTicket(cwData)
		ticket.AddedToStore = time.Now()
		if err := s.dataStore.UpsertTicket(ticket); err != nil {
			return nil, fmt.Errorf("inserting ticket into store: %w", err)
		}
	}

	return ticket, nil
}

// cwTicketToStoreTicket takes a Connectwise ticket info API response and converts it to a
// struct compatible with our data store.
func cwTicketToStoreTicket(cwData *cwData) *Ticket {
	return &Ticket{
		ID:        cwData.ticket.ID,
		Summary:   cwData.ticket.Summary,
		BoardID:   cwData.ticket.Board.ID,
		OwnerID:   cwData.ticket.Owner.ID,
		Resources: cwData.ticket.Resources,
		UpdatedBy: cwData.ticket.Info.UpdatedBy,
	}
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
