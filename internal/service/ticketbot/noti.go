package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
)

type message struct {
	webexMsg     webex.Message
	webexRoom    *models.WebexRoom
	notification models.TicketNotification
}

func newMessage(wm webex.Message, wr *models.WebexRoom, n models.TicketNotification) message {
	return message{
		webexMsg:     wm,
		webexRoom:    wr,
		notification: n,
	}
}

func (s *Service) ProcessWithNewTicket(ctx context.Context, ticket *models.FullTicket) error {
	if ticket == nil {
		return errors.New("received nil ticket")
	}

	notifiers, err := s.Notifiers.ListByBoard(ctx, ticket.Board.ID)
	if err != nil {
		return fmt.Errorf("listing notifiers for board: %w", err)
	}

	if len(notifiers) == 0 {
		return nil
	}

	var rooms []models.WebexRoom
	for _, n := range notifiers {
		r, err := s.Rooms.Get(ctx, n.WebexRoomID)
		if err != nil {
			return fmt.Errorf("getting webex room from notifier: %w", err)
		}

		rooms = append(rooms, r)
	}

	msgs := s.makeNewTicketMessages(rooms, ticket)
	var msgErrs []error

	for _, m := range msgs {
		if _, err := s.WebexClient.PostMessage(&m.webexMsg); err != nil {
			e := fmt.Errorf("sending webex message: %w", err)
			msgErrs = append(msgErrs, e)
		}

		n, err := s.Notifications.Insert(ctx, m.notification)
		if err != nil {
			e := fmt.Errorf("inserting notification into store: %w", err)
			msgErrs = append(msgErrs, e)
			continue
		}
		slog.Info("sent new ticket notification", "id", n.ID, "ticket_id", ticket.Ticket.ID, "to_webex_room", m.webexRoom.Name)
	}

	if len(msgErrs) > 0 {
		for _, e := range msgErrs {
			slog.Error("sending ticket notification", "error", e)
		}
		return fmt.Errorf("sending ticket notifications for ticket %d - see logs for details", ticket.Ticket.ID)
	}

	return nil
}

func (s *Service) makeNewTicketMessages(rooms []models.WebexRoom, ticket *models.FullTicket) []message {
	body := "**New Ticket:** "
	if ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", ticket.Company.Name)
	}

	if ticket.Contact != nil {
		name := fullName(ticket.Contact.FirstName, ticket.Contact.LastName)
		body += fmt.Sprintf("\n**Ticket Contact:** %s", name)
	}

	if ticket.LatestNote != nil && ticket.LatestNote.Content != nil {
		body += messageText(ticket, s.MaxMessageLength)
	}

	var msgs []message
	for _, r := range rooms {
		wm := webex.NewMessageToRoom(r.WebexID, r.Name, body)

		n := &models.TicketNotification{
			TicketID:    ticket.Ticket.ID,
			WebexRoomID: &r.ID,
			Sent:        true,
		}

		if ticket.LatestNote != nil {
			n.TicketNoteID = &ticket.LatestNote.ID
		}

		msgs = append(msgs, newMessage(wm, &r, *n))
	}

	return msgs
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
