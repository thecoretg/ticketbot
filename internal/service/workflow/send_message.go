package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/models"
)

// SendMessage sends a Webex message to a configured recipient (room or person).
// It is NOT idempotent (re-running posts a duplicate), so the engine guards it
// with run-once markers. When SkipFurtherNotifications is set and the message is
// sent, it signals the engine to suppress the downstream notifier for the ticket.
type SendMessage struct{}

type SendMessageParams struct {
	RecipientID int `json:"recipient_id"`
	// UseTicketCard sends the standard formatted ticket notification card (summary,
	// company, contact, latest note) instead of the custom Text body.
	UseTicketCard bool   `json:"use_ticket_card"`
	Text          string `json:"text" tmpl:"text"`
	// SkipFurtherNotifications suppresses the notifier for this ticket once the
	// message has been sent, so a workflow-driven notification doesn't double up
	// with the standard notification.
	SkipFurtherNotifications bool `json:"skip_further_notifications"`
}

func (*SendMessageParams) isParams() {}

func (SendMessage) ActionType() string { return models.WorkflowActionSendMessage }
func (SendMessage) NewParams() Params  { return &SendMessageParams{} }
func (SendMessage) Idempotent() bool   { return false }

func (SendMessage) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*SendMessageParams)
	if pp.RecipientID == 0 {
		return Change{}, fmt.Errorf("send_message: no recipient configured")
	}

	r, err := x.Recips.Get(ctx, pp.RecipientID)
	if err != nil {
		return Change{}, fmt.Errorf("resolving recipient %d: %w", pp.RecipientID, err)
	}

	body := pp.Text
	if pp.UseTicketCard {
		body = ticketCard(t, x.LastNote, x.CWCompanyID, x.MaxMessageLength)
	}
	if strings.TrimSpace(body) == "" {
		return Change{Field: "message"}, nil // nothing to send
	}

	if x.Simulate {
		return Change{Applied: true, Field: "message", To: r.Name, SkipNotify: pp.SkipFurtherNotifications}, nil
	}

	msg := newWebexMsg(r, body)
	if _, err := x.Webex.PostMessage(ctx, &msg); err != nil {
		return Change{}, fmt.Errorf("sending webex message to %q: %w", r.Name, err)
	}

	return Change{Applied: true, Field: "message", To: r.Name, SkipNotify: pp.SkipFurtherNotifications}, nil
}

// newWebexMsg builds a Webex message addressed to a person (by email) or a room.
func newWebexMsg(r *models.WebexRecipient, body string) webex.Message {
	if r.Type == models.RecipientTypePerson && r.Email != nil {
		return webex.NewMessageToPerson(*r.Email, body)
	}
	return webex.NewMessageToRoom(r.WebexID, r.Name, body)
}

// ticketCard composes the standard ticket notification body from the live ticket
// and its most recent note, mirroring the notifier's message layout. It works
// from the CW ticket (pre-sync) rather than the synced FullTicket.
func ticketCard(t *psa.Ticket, note *psa.ServiceTicketNote, companyID string, maxLen int) string {
	body := fmt.Sprintf("**Ticket:** %s %s", psa.MarkdownInternalTicketLink(t.ID, companyID), t.Summary)

	if t.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", t.Company.Name)
	}
	if t.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", t.Contact.Name)
	}

	if note != nil && note.Text != "" {
		sender := note.Member.Name
		if sender == "" {
			sender = note.Contact.Name
		}
		if sender != "" {
			body += fmt.Sprintf("\n**Latest Note Sent By:** %s", sender)
		}

		content := note.Text
		if maxLen > 0 && len(content) > maxLen {
			content = content[:maxLen] + "..."
		}
		body += fmt.Sprintf("\n%s", blockQuote(content))
	}

	body += "\n\n---"
	return body
}

// blockQuote renders text as a markdown block quote, preserving line breaks.
func blockQuote(text string) string {
	var b strings.Builder
	for i, line := range strings.Split(text, "\n") {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("> ")
		b.WriteString(line)
	}
	return b.String()
}
