package notifier

import (
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type Message struct {
	MsgType      string
	WebexMsg     webex.Message
	WebexRoom    models.WebexRecipient
	Notification models.TicketNotification
	SendError    error
}

func newMessage(msgType string, wm webex.Message, wr models.WebexRecipient, n models.TicketNotification) Message {
	return Message{
		MsgType:      msgType,
		WebexMsg:     wm,
		WebexRoom:    wr,
		Notification: n,
	}
}

func (s *Service) makeTicketMessages(t *models.FullTicket, recips []models.WebexRecipient, isNew bool) []Message {
	if isNew {
		return s.makeNewTicketMessages(t, recips)
	}

	return s.makeUpdatedTicketMessages(t, recips)
}

func (s *Service) makeNewTicketMessages(t *models.FullTicket, recips []models.WebexRecipient) []Message {
	header := fmt.Sprintf("**New Ticket:** %s %s", psa.MarkdownInternalTicketLink(t.Ticket.ID, s.CWCompanyID), t.Ticket.Summary)
	body := makeMessageBody(t, header, s.MaxMessageLength)

	var msgs []Message
	for _, r := range recips {
		wm := webex.NewMessageToRoom(r.WebexID, r.Name, body)

		n := &models.TicketNotification{
			TicketID:    t.Ticket.ID,
			RecipientID: r.ID,
			Sent:        true,
		}

		if t.LatestNote != nil {
			n.TicketNoteID = &t.LatestNote.ID
		}

		msgs = append(msgs, newMessage("new_ticket", wm, r, *n))
	}

	return msgs
}

func (s *Service) makeUpdatedTicketMessages(t *models.FullTicket, recips []models.WebexRecipient) []Message {
	header := fmt.Sprintf("**Ticket Updated:** %s %s", psa.MarkdownInternalTicketLink(t.Ticket.ID, s.CWCompanyID), t.Ticket.Summary)
	body := makeMessageBody(t, header, s.MaxMessageLength)

	var msgs []Message
	for _, r := range recips {
		wm := webex.NewMessageToRoom(r.WebexID, r.Name, body)

		n := &models.TicketNotification{
			TicketID:    t.Ticket.ID,
			RecipientID: r.ID,
			Sent:        true,
		}

		if t.LatestNote != nil {
			n.TicketNoteID = &t.LatestNote.ID
		}

		msgs = append(msgs, newMessage("updated_ticket", wm, r, *n))
	}

	return msgs
}

func makeMessageBody(ticket *models.FullTicket, header string, maxLen int) string {
	body := header
	if ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", ticket.Company.Name)
	}

	if ticket.Contact != nil {
		name := fullName(ticket.Contact.FirstName, ticket.Contact.LastName)
		body += fmt.Sprintf("\n**Ticket Contact:** %s", name)
	}

	if ticket.LatestNote != nil && ticket.LatestNote.Content != nil {
		body += messageText(ticket, maxLen)
	}

	// Divider line for easily distinguishable breaks in notifications
	body += "\n\n---"
	return body
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
