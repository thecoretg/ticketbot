package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"tctg-automation/pkg/connectwise"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *server) addHooksGroup() {
	hooks := s.ginEngine.Group("/hooks")
	cw := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.config.ExitOnError))
	cw.POST("/tickets", s.handleTickets)
}

func (s *server) handleTickets(c *gin.Context) {
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
		if err := s.addOrUpdateTicket(c.Request.Context(), w.ID, w.Action, true); err != nil {
			c.Error(fmt.Errorf("adding or updating the ticket into data storage: %w", err))
			return
		}

		c.Status(http.StatusNoContent)
	} else {
		c.Status(http.StatusNoContent)
	}
}

func (s *server) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := s.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

// addOrUpdateTicket serves as the primary handler for updating the data store with ticket data. It also will handle
// extra functionality such as ticket notifications.
func (s *server) addOrUpdateTicket(ctx context.Context, ticketID int, action string, attemptNotify bool) error {
	// Lock the ticket so that extra calls don't interfere. Due to the nature of Connectwise updates will often
	// result in other hooks and actions taking place, which means a ticket rarely only sends one webhook payload.
	lock := s.getTicketLock(ticketID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	// Get existing ticket from store - will be nil if it doesn't already exist.
	storeTicket, err := s.dataStore.GetTicket(ticketID)
	if err != nil {
		return fmt.Errorf("getting ticket from storage: %w", err)
	}

	// Get the current data for the ticket via the Connectwise API.
	// This will be used to compare for changes with the store ticket.
	cwTicketData, err := s.cwClient.GetTicket(ticketID, nil)
	if err != nil {
		return fmt.Errorf("getting ticket data from connectwise: %w", err)
	}

	// Get the board the ticket's in from the store - will be nil if it doesn't already exist.
	board, err := s.dataStore.GetBoard(cwTicketData.Board.ID)
	if err != nil {
		return fmt.Errorf("getting board from storage: %w", err)
	}

	// If the board is nil, add it to the store.
	if board == nil {
		board, err = s.addBoard(cwTicketData.Board.ID)
		if err != nil {
			return err
		}
		slog.Info("added board to store", "board_id", board.ID, "name", board.Name)
	}

	// Get the most recent note from the ticket. This will be used for the notifier.
	lastNote, err := s.getLatestNote(cwTicketData.ID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	// Convert the ticket data from Connectwise into a store-compatible ticket.
	// If the store ticket is nil, we'll add the current time as the time it was added.
	workingTicket := cwTicketToStoreTicket(cwTicketData, lastNote)
	if storeTicket != nil {
		workingTicket.AddedToStore = storeTicket.AddedToStore
	} else {
		workingTicket.AddedToStore = time.Now()
	}

	// Compare the store ticket and the working ticket to see if there are differences.
	// Also check if the most recent note counts as new for notifier purposes.
	ticketChanged, changeList := findChanges(storeTicket, workingTicket)
	noteChanged := noteCountsAsNew(storeTicket, lastNote)

	// Log all relevant info if debug is enabled.
	slog.Debug(
		"ticket checked for changes", "action", action, "ticket_id", workingTicket.ID, "changes_found", changeList,
		"note_changed", noteChanged, "latest_note_id", workingTicket.LatestNoteID, "attempt_notify", attemptNotify, "board_notify_enabled", board.NotifyEnabled,
	)

	// Insert or update the ticket into the store if it didn't exist or if there were changes.
	if ticketChanged {
		if err := s.dataStore.UpsertTicket(workingTicket); err != nil {
			return fmt.Errorf("upserting ticket to store: %w", err)
		}
	}

	// Use the action from the CW hook, whether the note is considered new, and if the board
	// has notifications enabled to determine what type of notification will be sent, if any.
	if meetsMessageCriteria(action, noteChanged, board) {
		// Create the Webex messages. Depending on arguments, it will create either a message to attached resources,
		// or it will create a new ticket alert for a Webex room.
		messages, err := s.makeWebexMsgs(action, cwTicketData.Info.UpdatedBy, board, cwTicketData, lastNote)
		if err != nil {
			return fmt.Errorf("making webex messages: %w", err)
		}

		// Loop through all of the created messages and send them via Webex.
		for _, msg := range messages {
			if err := s.webexClient.SendMessage(ctx, msg); err != nil {
				// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
				// the person on the ticket doesn't have an account, or the same email address, in Webex.
				slog.Warn("error sending webex message", "ticket_id", workingTicket.ID, "room_id", msg.RoomId, "person", msg.Person, "error", err)
			}
		}
	}

	return nil
}

func (s *server) getLatestNote(ticketID int) (*connectwise.ServiceTicketNote, error) {
	note, err := s.cwClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note from connectwise: %w", err)
	}

	if note == nil {
		note = &connectwise.ServiceTicketNote{}
	}

	return note, nil
}

// addBoard adds Connecwise boards to the data store, with a default of
// notifications not enabled.
func (s *server) addBoard(boardID int) (*Board, error) {
	cwBoard, err := s.cwClient.GetBoard(boardID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting board from connectwise: %w", err)
	}

	storeBoard := &Board{
		ID:            cwBoard.ID,
		Name:          cwBoard.Name,
		NotifyEnabled: false,
	}

	if err := s.dataStore.UpsertBoard(storeBoard); err != nil {
		return nil, fmt.Errorf("adding board to store: %w", err)
	}

	return storeBoard, nil
}

// cwTicketToStoreTicket takes a Connectwise ticket info API response and converts it to a
// struct compatible with our data store.
func cwTicketToStoreTicket(cwTicket *connectwise.Ticket, latestNote *connectwise.ServiceTicketNote) *Ticket {
	return &Ticket{
		ID:           cwTicket.ID,
		Summary:      cwTicket.Summary,
		BoardID:      cwTicket.Board.ID,
		LatestNoteID: latestNote.ID,
		OwnerID:      cwTicket.Owner.ID,
		Resources:    cwTicket.Resources,
		UpdatedBy:    cwTicket.Info.UpdatedBy,
	}
}

// findChanges compares fields in two data store tickets. It returns a bool for if changes were detected,
// and a comma-separated string of the changes it found (or "none")
func findChanges(a, b *Ticket) (bool, string) {
	if a == nil || b == nil {
		return a != b, "one of the tickets is nil"
	}

	var changedValues []string
	if a.ID != b.ID {
		changedValues = append(changedValues, "ID")
	}

	if a.Summary != b.Summary {
		changedValues = append(changedValues, "Summary")
	}

	if a.BoardID != b.BoardID {
		changedValues = append(changedValues, "BoardID")
	}

	if a.LatestNoteID != b.LatestNoteID {
		changedValues = append(changedValues, "LatestNoteID")
	}

	if a.OwnerID != b.OwnerID {
		changedValues = append(changedValues, "OwnerID")
	}

	if a.UpdatedBy != b.UpdatedBy {
		changedValues = append(changedValues, "UpdatedBy")
	}

	if a.Resources != b.Resources {
		changedValues = append(changedValues, "Resources")
	}

	changeStr := "none"
	if len(changedValues) > 0 {
		changeStr = strings.Join(changedValues, ", ")
	}
	return len(changedValues) > 0, changeStr
}

// noteCountsAsNew takes a store ticket and a note, and determines if the note is to be considered
// new for notification purposes.
func noteCountsAsNew(storeTicket *Ticket, lastNote *connectwise.ServiceTicketNote) bool {
	b := false

	// first, check if the store ticket exists and if the stored note ID doesn't match the ID of the latest note.
	if storeTicket != nil && storeTicket.LatestNoteID != lastNote.ID {
		b = true
		// then, check if the stored ticket's note ID is HIGHER than that of the latest note.
		// why? If it is higher, that means the stored ticket's latest note was likely deleted.
		// In cases like these, we don't want to notify resources because it would mean the latest note
		// is one they probably got notified for previously, hence appearing to be a duplicate.
		if storeTicket.LatestNoteID > lastNote.ID {
			b = false // dont notify about a note they probably already got notified for
		}
	}

	return b
}

// meetsMessageCriteria checks if a message would be allowed to send a notification,
// depending on if it was added or updated, if the note changed, and the board's notification settings.
func meetsMessageCriteria(action string, noteChanged bool, board *Board) bool {
	if action == "added" {
		return board.NotifyEnabled
	}

	if action == "updated" {
		return noteChanged && board.NotifyEnabled
	}

	return false
}
