package ticketbot

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"sync"
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

		if err := s.addOrUpdateTicket(w.Action, storeTicket, cwTicket, true); err != nil {
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

func (s *server) addOrUpdateTicket(action string, storeTicket *Ticket, cwTicket *connectwise.Ticket, attemptNotify bool) error {
	lock := s.getTicketLock(cwTicket.ID)
	if !lock.TryLock() {
		lock.Lock()
	}

	defer func() {
		lock.Unlock()
	}()

	board, err := s.dataStore.GetBoard(cwTicket.Board.ID)
	if err != nil {
		return fmt.Errorf("getting board from storage: %w", err)
	}

	if board == nil {
		board, err = s.addBoard(cwTicket.Board.ID)
		if err != nil {
			return err
		}
		slog.Info("added board to store", "board_id", board.ID, "name", board.Name)
	}

	lastNote, err := s.getLatestNote(cwTicket.ID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	newTicket := cwTicketToStoreTicket(cwTicket, lastNote)
	if storeTicket != nil {
		newTicket.AddedToStore = storeTicket.AddedToStore
	} else {
		newTicket.AddedToStore = time.Now()
	}

	ticketChanged, changeList := findChanges(storeTicket, newTicket)
	noteChanged := false
	if storeTicket != nil && storeTicket.LatestNoteID != lastNote.ID {
		noteChanged = true
		if storeTicket.LatestNoteID > lastNote.ID {
			slog.Warn("latest note id in store is greater than latest from connectwise -  the note was likely deleted", "ticket_id", newTicket.ID, "store_note_id", storeTicket.LatestNoteID, "latest_note_id", lastNote.ID)
			noteChanged = false // dont notify about a note they probably already got notified for
		}
	}

	slog.Debug(
		"ticket checked for changes", "action", action, "ticket_id", newTicket.ID, "changes_found", changeList,
		"note_changed", noteChanged, "latest_note_id", newTicket.LatestNoteID, "attempt_notify", attemptNotify, "board_notify_enabled", board.NotifyEnabled,
	)

	if ticketChanged {
		if err := s.dataStore.UpsertTicket(newTicket); err != nil {
			return fmt.Errorf("upserting ticket to store: %w", err)
		}
	}

	// if latest note changed, proceed with notification
	if noteChanged && attemptNotify && board.NotifyEnabled {
		// this will do stuff once implemented
		//msg := makeWebexMsg(action, s.config.CWCreds.CompanyId, sendTo, cwTicket, lastNote, s.config.MaxMsgLength)
		//if err := s.webexClient.SendMessage(ctx, msg); err != nil {
		//	return fmt.Errorf("sending webex message: %w", err)
		//}
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
		//WebexRooms:    []WebexRoom{},
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
