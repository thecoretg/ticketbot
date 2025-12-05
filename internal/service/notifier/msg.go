package notifier

import (
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type Message struct {
	MsgType        string
	WebexMsg       webex.Message
	WebexRecipient recipData
	Notification   models.TicketNotification
	SendError      error
}

func newMessage(wm webex.Message, r recipData, n models.TicketNotification, isNew bool) Message {
	mt := "updated_ticket"
	if isNew {
		mt = "new_ticket"
	}

	return Message{
		MsgType:        mt,
		WebexMsg:       wm,
		WebexRecipient: r,
		Notification:   n,
	}
}

func (s *Service) makeTicketMessages(t *models.FullTicket, recips []recipData, isNew bool) []Message {
	mainHeader := s.notificationHeader(t, isNew)

	var msgs []Message
	for _, r := range recips {
		var h string
		if !r.isNaturalRecipient() {
			h += fmt.Sprintf("%s\n", fwdChainStr(r.recipient, r.forwardChain))
		}

		h += mainHeader
		body := makeMessageBody(t, h, s.MaxMessageLength)

		wm := newWebexMsg(r.recipient, body)
		n := &models.TicketNotification{
			TicketID:    t.Ticket.ID,
			RecipientID: r.recipient.ID,
			Sent:        true,
		}

		if t.LatestNote != nil {
			n.TicketNoteID = &t.LatestNote.ID
		}

		if r.forwardChain != nil {
			n.ForwardedFromID = &r.forwardChain[len(r.forwardChain)-1].ID
		}

		msgs = append(msgs, newMessage(wm, r, *n, isNew))
	}

	return msgs
}

func fwdChainStr(recip models.WebexRecipient, fwdChain []models.WebexRecipient) string {
	names := make([]string, 0, len(fwdChain))
	for _, f := range fwdChain {
		names = append(names, f.Name)
	}

	rn := "You"
	if recip.Type == models.RecipientTypeRoom {
		rn = recip.Name
	}

	ch := strings.Join(names, " > ")
	return fmt.Sprintf("**FWD:** %s > %s", ch, rn)
}

func (s *Service) notificationHeader(t *models.FullTicket, isNew bool) string {
	if isNew {
		return fmt.Sprintf("**New Ticket:** %s %s", psa.MarkdownInternalTicketLink(t.Ticket.ID, s.CWCompanyID), t.Ticket.Summary)
	}

	return fmt.Sprintf("**Ticket Updated:** %s %s", psa.MarkdownInternalTicketLink(t.Ticket.ID, s.CWCompanyID), t.Ticket.Summary)
}

func newWebexMsg(r models.WebexRecipient, body string) webex.Message {
	if r.Type == models.RecipientTypePerson && r.Email != nil {
		return webex.NewMessageToPerson(*r.Email, body)
	}

	return webex.NewMessageToRoom(r.WebexID, r.Name, body)
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
