package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	ErrNoRooms     = errors.New("no rooms enabled for this board")
	ErrNoNote      = errors.New("no note found for this ticket")
	ErrAlreadySent = errors.New("notification(s) already sent for this note")
)

func (s *Service) CreateMessages(ctx context.Context, ticket *models.FullTicket, action, cwClientID string) error {
	if ticket == nil {
		return errors.New("received nil ticket")
	}

	// return ErrNoNote so caller can check (and return non-error)
	noteID := ticket.LatestNote.ID
	if noteID == 0 {
		return ErrNoNote
	}

	rooms, err := s.Notifiers.ListByBoard(ctx, ticket.Board.ID)
	if err != nil {
		return fmt.Errorf("getting enabled rooms: %w", err)
	}

	if rooms == nil || len(rooms) == 0 {
		return ErrNoRooms
	}

	sent, err := s.Notifications.ExistsForNote(ctx, noteID)
	if err != nil {
		return fmt.Errorf("checking if notification exists for note: %w", err)
	}

	if sent {
		return ErrAlreadySent
	}

}

func (s *Service) MarkNotiSkipped(ctx context.Context, ticket *models.FullTicket) error {
	if ticket.LatestNote == nil {
		return nil
	}

	noteID := ticket.LatestNote.ID
	notified, err := s.Notifications.ExistsForNote(ctx, noteID)
	if err != nil {
		return fmt.Errorf("checking existence of notification for note: %w", err)
	}

	if notified {
		return nil
	}

	n := models.TicketNotification{
		TicketNoteID: noteID,
		Skipped:      true,
	}

	if _, err := s.Notifications.Insert(ctx, n); err != nil {
		return fmt.Errorf("inserting notification: %w", err)
	}

	return nil
}

func (s *Service) makeMessages(ctx context.Context, action string, t *models.FullTicket) ([]webex.Message, error) {
	var body string
	body += messageHeader(t, action, s.CWCompanyID)

	// add company name if present (even Catchall is considered a company; this will always exist)
	if t.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", t.Company.Name)
	}

	// add ticket contact name if exists (not always true)
	if t.Contact.ID != 0 {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", fullName(t.Contact.FirstName, t.Contact.LastName))
	}

	if rs.cwData.note.Text != "" {
		body += cl.messageText(rs.cwData)
	}
}

func messageHeader(t *models.FullTicket, action, cwClientID string) string {
	var header string
	if action == "added" {
		header += "**New Ticket:** "
	} else {
		header += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	header += fmt.Sprintf("%s %s", psa.MarkdownInternalTicketLink(t.Ticket.ID, cwClientID), t.Ticket.Summary)
	return header
}

func messageText(t *models.FullTicket, maxLen int) string {
	var body string
	sender := getSenderName(t)
	if sender != "" {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", sender)
	}

	content := ""
	if t.LatestNote.Content != nil {
		content = *t.LatestNote.Content
	}

	if len(content) > maxLen {
		content = content[:maxLen] + "..."
	}
	body += fmt.Sprintf("\n%s", blockQuoteText(content))
	return body
}

// blockQuoteText creates a markdown block quote from a string, also respects line breaks
func blockQuoteText(text string) string {
	parts := strings.Split(text, "\n")
	for i, part := range parts {
		parts[i] = "> " + part
	}

	return strings.Join(parts, "\n")
}

// getSenderName determines the name of the sender of a note. It checks for members in Connectwise and external contacts from companies.
func getSenderName(t *models.FullTicket) string {
	if t.LatestNote.Member != nil {
		return fullName(t.LatestNote.Member.FirstName, &t.LatestNote.Member.LastName)
	} else if t.LatestNote.Contact != nil {
		return fullName(t.LatestNote.Contact.FirstName, t.LatestNote.Contact.LastName)
	}

	return ""
}

func fullName(first string, last *string) string {
	if last != nil {
		return fmt.Sprintf("%s %s", first, *last)
	}

	return first
}
