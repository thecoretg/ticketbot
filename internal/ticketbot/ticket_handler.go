package ticketbot

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"
	"sync"
	"tctg-automation/internal/ticketbot/types"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
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
		// check if ticket already exists in store

		storeTicket, err := s.dataStore.GetTicket(w.ID)
		if err != nil {
			c.Error(fmt.Errorf("getting ticket from storage: %w", err))
			return
		}

		cwTicket, err := s.cwClient.GetTicket(w.ID, nil)
		if err != nil {
			c.Error(fmt.Errorf("getting ticket from connectwise: %w", err))
			return
		}

		if err := s.addOrUpdateTicket(storeTicket, cwTicket, true); err != nil {
			c.Error(fmt.Errorf("adding or updating the ticket into data storage: %w", err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *server) getTicketLock(ticketID int) *sync.Mutex {
	lockIface, _ := s.ticketLocks.LoadOrStore(ticketID, &sync.Mutex{})
	return lockIface.(*sync.Mutex)
}

func (s *server) addOrUpdateTicket(storeTicket *types.Ticket, cwTicket *connectwise.Ticket, attemptNotify bool) error {
	lock := s.getTicketLock(cwTicket.ID)
	if !lock.TryLock() {
		slog.Debug("waiting for ticket lock to resolve", "ticket_id", cwTicket.ID)
		lock.Lock()
	}

	slog.Debug("locked ticket", "ticket_id", cwTicket.ID)
	defer func() {
		lock.Unlock()
		slog.Debug("unlocked ticket", "ticket_id", cwTicket.ID)
	}()

	board, err := s.dataStore.GetBoard(cwTicket.Board.ID)
	if err != nil {
		return fmt.Errorf("getting board from storage: %w", err)
	}

	if board == nil {
		slog.Debug("no board found in store", "board_id", cwTicket.Board.ID)
		board, err = s.addBoard(cwTicket.Board.ID)
		if err != nil {
			return err
		}
		slog.Info("added board to store", "board_id", board.ID, "name", board.Name)
	} else {
		slog.Debug("found board in store", "board_id", board.ID, "name", board.Name)
	}

	lastNote, err := s.getLatestNote(cwTicket.ID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	newTicket := cwTicketToStoreTicket(cwTicket, lastNote)
	if storeTicket != nil {
		slog.Debug("got ticket from store", "ticket_id", storeTicket.ID, "summary", storeTicket.Summary, "latest_note_id", storeTicket.LatestNoteID)
		newTicket.AddedToStore = storeTicket.AddedToStore
	} else {
		slog.Debug("no ticket found in store", "ticket_id", cwTicket.ID)
		newTicket.AddedToStore = time.Now()
	}

	ticketChanged := !reflect.DeepEqual(storeTicket, newTicket)
	noteChanged := false
	if storeTicket != nil && storeTicket.LatestNoteID != lastNote.ID {
		noteChanged = true
	}

	if ticketChanged {
		slog.Debug("found changes for ticket", "ticket_id", newTicket.ID)
		if err := s.dataStore.UpsertTicket(newTicket); err != nil {
			return fmt.Errorf("upserting ticket to store: %w", err)
		}
		slog.Info("upserted ticket to storage", "ticket_id", newTicket.ID, "summary", newTicket.Summary, "latest_note_id", newTicket.LatestNoteID)
	} else {
		slog.Debug("no changes found for ticket", "ticket_id", newTicket.ID)
		return nil
	}

	// if latest note changed, proceed with notification
	if noteChanged && attemptNotify {
		slog.Debug("found note change", "ticket_id", newTicket.ID, "old_note", newTicket.LatestNoteID, "new_note", lastNote.ID)
		if board.NotifyEnabled {
			slog.Debug("note changed, running notifier", "ticket_id", newTicket.ID, "note_id", newTicket.LatestNoteID, "board_id", board.ID)
			// this will do stuff once implemented
			//msg := makeWebexMsg(action, s.config.CWCreds.CompanyId, sendTo, cwTicket, lastNote, s.config.MaxMsgLength)
			//if err := s.webexClient.SendMessage(ctx, msg); err != nil {
			//	return fmt.Errorf("sending webex message: %w", err)
			//}
		} else {
			slog.Debug("note changed, but board notify not enabled", "ticket_id", newTicket.ID, "board_id", board.ID)
		}
	} else {
		slog.Debug("note did not change", "ticket_id", newTicket.ID, "old_note", newTicket.LatestNoteID, "new_note", lastNote.ID)
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

func (s *server) addBoard(boardID int) (*types.Board, error) {
	cwBoard, err := s.cwClient.GetBoard(boardID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting board from connectwise: %w", err)
	}

	storeBoard := &types.Board{
		ID:            cwBoard.ID,
		Name:          cwBoard.Name,
		NotifyEnabled: false,
		WebexRoomIDs:  nil,
	}

	if err := s.dataStore.UpsertBoard(storeBoard); err != nil {
		return nil, fmt.Errorf("adding board to store: %w", err)
	}

	return storeBoard, nil
}

func cwTicketToStoreTicket(cwTicket *connectwise.Ticket, latestNote *connectwise.ServiceTicketNote) *types.Ticket {
	return &types.Ticket{
		ID:           cwTicket.ID,
		Summary:      cwTicket.Summary,
		BoardID:      cwTicket.Board.ID,
		LatestNoteID: latestNote.ID,
		OwnerID:      cwTicket.Owner.ID,
		Resources:    cwTicket.Resources,
		UpdatedBy:    cwTicket.Info.UpdatedBy,
		TimeDetails: types.TimeDetails{
			UpdatedAt: cwTicket.Info.LastUpdated,
		},
	}
}

// getSendTo creates a list of emails to send notifications to, factoring in who made the most
// recent update and any other exclusions passed in by the config.
func (s *server) getSendTo(updatedBy string, ticket *connectwise.Ticket) ([]string, error) {
	var excludedMembers []string
	for _, m := range s.config.ExcludedCWMembers {
		excludedMembers = append(excludedMembers, m)
	}

	if ticket.Info.UpdatedBy != "" {
		excludedMembers = append(excludedMembers, updatedBy)
	}

	identifiers := filterOutExcluded(excludedMembers, ticket.Resources)
	condition := fmt.Sprintf("identifier in (%s)", identifiers)

	params := map[string]string{
		"conditions": condition,
	}

	// get members from connectwise and then create a list of emails
	members, err := s.cwClient.ListMembers(params)
	if err != nil {
		return nil, fmt.Errorf("getting members from connectwise: %w", err)
	}

	var emails []string
	for _, m := range members {
		if m.PrimaryEmail != "" {
			emails = append(emails, m.PrimaryEmail)
		}
	}

	return emails, nil
}

func makeWebexMsg(action, companyID, sendTo string, ticket *connectwise.Ticket, note *connectwise.ServiceTicketNote, maxLength int) webex.MessagePostBody {
	text := note.Text
	if len(text) > maxLength {
		text = text[:maxLength] + "..."
	}

	var body string
	if action == "added" {
		body += "**New Ticket:** "
	} else {
		body += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	body += fmt.Sprintf("%s %s", connectwise.MarkdownInternalTicketLink(ticket.ID, companyID), ticket.Summary)

	// add company name if present (even Catchall is considered a company, so this will always exist)
	if ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", ticket.Company.Name)
	}

	// add ticket contact name if exists (not always true)
	if ticket.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", ticket.Contact.Name)
	}

	if note.Text != "" {
		sender := getSenderName(note)
		if sender != nil {
			body += fmt.Sprintf("\n**Latest Note Sent By:** %s", *sender)
		}

		body += fmt.Sprintf("\n%s", blockQuoteText(note.Text))
	}

	if action == "added" {
		return webex.NewMessageToRoom(sendTo, body)
	} else {
		return webex.NewMessageToPerson(sendTo, body)
	}
}

func getSenderName(note *connectwise.ServiceTicketNote) *string {
	if note.Member.Name != "" {
		return &note.Member.Name
	} else if note.CreatedBy != "" {
		return &note.CreatedBy
	} else if note.Contact.Name != "" {
		return &note.Contact.Name
	}

	return nil
}

func filterOutExcluded(excluded []string, identifiers string) string {
	var parts []string
	for _, i := range strings.Split(identifiers, ",") {
		if !slices.Contains(excluded, i) {
			parts = append(parts, i)
		}
	}

	return strings.Join(parts, ",")
}

// blockQuoteText creates a markdown block quote from a string, also respects line breaks
func blockQuoteText(text string) string {
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		parts[i] = "> " + part
	}

	return strings.Join(parts, "\n")
}
