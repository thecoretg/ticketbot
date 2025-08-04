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

func (s *server) addOrUpdateTicket(ctx context.Context, ticketID int, action string, attemptNotify bool) error {
	lock := s.getTicketLock(ticketID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	storeTicket, err := s.dataStore.GetTicket(ticketID)
	if err != nil {
		return fmt.Errorf("getting ticket from storage: %w", err)
	}

	cwTicketData, err := s.cwClient.GetTicket(ticketID, nil)
	if err != nil {
		return fmt.Errorf("getting ticket data from connectwise: %w", err)
	}

	board, err := s.dataStore.GetBoard(cwTicketData.Board.ID)
	if err != nil {
		return fmt.Errorf("getting board from storage: %w", err)
	}

	if board == nil {
		board, err = s.addBoard(cwTicketData.Board.ID)
		if err != nil {
			return err
		}
		slog.Info("added board to store", "board_id", board.ID, "name", board.Name)
	}

	lastNote, err := s.getLatestNote(cwTicketData.ID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	workingTicket := cwTicketToStoreTicket(cwTicketData, lastNote)
	if storeTicket != nil {
		workingTicket.AddedToStore = storeTicket.AddedToStore
	} else {
		workingTicket.AddedToStore = time.Now()
	}

	ticketChanged, changeList := findChanges(storeTicket, workingTicket)
	noteChanged := noteCountsAsNew(storeTicket, lastNote)

	slog.Debug(
		"ticket checked for changes", "action", action, "ticket_id", workingTicket.ID, "changes_found", changeList,
		"note_changed", noteChanged, "latest_note_id", workingTicket.LatestNoteID, "attempt_notify", attemptNotify, "board_notify_enabled", board.NotifyEnabled,
	)

	if ticketChanged {
		if err := s.dataStore.UpsertTicket(workingTicket); err != nil {
			return fmt.Errorf("upserting ticket to store: %w", err)
		}
	}

	if meetsMessageCriteria(action, noteChanged, board) {
		messages, err := s.makeWebexMsgs(action, cwTicketData.Info.UpdatedBy, board, cwTicketData, lastNote)
		if err != nil {
			return fmt.Errorf("making webex messages: %w", err)
		}

		for _, msg := range messages {
			if err := s.webexClient.SendMessage(ctx, msg); err != nil {
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

func noteCountsAsNew(storeTicket *Ticket, lastNote *connectwise.ServiceTicketNote) bool {
	b := false

	if storeTicket != nil && storeTicket.LatestNoteID != lastNote.ID {
		b = true
		if storeTicket.LatestNoteID > lastNote.ID {
			b = false // dont notify about a note they probably already got notified for
		}
	}

	return b
}

func meetsMessageCriteria(action string, noteChanged bool, board *Board) bool {
	if action == "added" {
		return board.NotifyEnabled
	}

	if action == "updated" {
		return noteChanged && board.NotifyEnabled
	}

	return false
}
