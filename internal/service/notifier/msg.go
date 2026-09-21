package notifier

import (
	"fmt"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/msgtemplate"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type Message struct {
	MsgType        string
	WebexMsg       webex.Message
	WebexRecipient recipData
	Notification   *models.TicketNotification
	SendError      error
}

func newMessage(wm webex.Message, r recipData, n *models.TicketNotification, isNew bool) Message {
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

// makeTicketMessages builds one Webex message per recipient. intents maps a natural recipient id
// to the notify intent that named it; a forwarded recipient uses its origin's intent, so a rule's
// custom message follows the notification through forwards.
func (s *Service) makeTicketMessages(t *models.FullTicket, recips []recipData, isNew bool, intents map[int]workflow.NotifyIntent) []Message {
	var msgs []Message
	for _, r := range recips {
		in := intents[r.origin().ID]

		var body string
		if !r.isNaturalRecipient() {
			body = fwdChainStr(r.recipient, r.forwardChain) + "\n"
		}
		body += s.messageBody(t, in, isNew)

		wm := newWebexMsg(r.recipient, body)
		n := &models.TicketNotification{
			TicketID:    t.Ticket.ID,
			RecipientID: &r.recipient.ID,
			Sent:        true,
		}

		if t.LatestNote != nil {
			n.TicketNoteID = &t.LatestNote.ID
		}

		if r.forwardChain != nil {
			n.ForwardedFromID = &r.forwardChain[len(r.forwardChain)-1].ID
		}

		msgs = append(msgs, newMessage(wm, r, n, isNew))
	}

	return msgs
}

// messageBody renders the step's custom message when it has one, or the default layout.
func (s *Service) messageBody(t *models.FullTicket, in workflow.NotifyIntent, isNew bool) string {
	if tpl := strings.TrimSpace(in.Target.Message); tpl != "" {
		rendered := msgtemplate.Render(tpl, msgtemplate.Context{
			Ticket:     t,
			StepTitle:  in.Step.Title,
			IsNew:      isNew,
			CompanyID:  s.CWCompanyID,
			MaxNoteLen: s.Cfg.MaxMessageLength,
		})
		return rendered + "\n\n---"
	}

	return makeMessageBody(t, s.notificationHeader(t, isNew), s.Cfg.MaxMessageLength)
}

func fwdChainStr(recip *models.WebexRecipient, fwdChain []*models.WebexRecipient) string {
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

func newWebexMsg(r *models.WebexRecipient, body string) webex.Message {
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
		body += fmt.Sprintf("\n**Ticket Contact:** %s", msgtemplate.FullName(ticket.Contact.FirstName, ticket.Contact.LastName))
	}

	if q := msgtemplate.NoteQuote(ticket, maxLen); q != "" {
		body += "\n" + q
	}

	// Divider line for easily distinguishable breaks in notifications
	body += "\n\n---"
	return body
}
