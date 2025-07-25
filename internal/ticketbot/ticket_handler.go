package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"tctg-automation/internal/ticketbot/types"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *server) addTicketGroup(r *gin.Engine) {
	tickets := r.Group("/hooks")
	cw := tickets.Group("/cw", requireValidCWSignature(), ErrorHandler(s.config.ExitOnError))
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

		if err := s.addOrUpdateTicket(c.Request.Context(), storeTicket, cwTicket); err != nil {
			c.Error(fmt.Errorf("adding or updating the ticket into data storage: %w", err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *server) addOrUpdateTicket(ctx context.Context, storeTicket *types.Ticket, cwTicket *connectwise.Ticket) error {
	var t *types.Ticket
	if storeTicket != nil {
		t = storeTicket
	} else {
		t = &types.Ticket{
			ID:      cwTicket.ID,
			Summary: cwTicket.Summary,
			TimeDetails: types.TimeDetails{
				UpdatedAt: time.Now(),
			},
		}
	}

	lastNote, err := s.cwClient.GetMostRecentTicketNote(cwTicket.ID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	if t.LatestNoteID != lastNote.ID {
		// update store early in case of multiple requests for the same ticket
		t.LatestNoteID = lastNote.ID
		t.UpdatedAt = time.Now()
		if err := s.dataStore.UpsertTicket(t); err != nil {
			return fmt.Errorf("upserting ticket to store: %w", err)
		}

		//msg := makeWebexMsg(action, s.config.CWCreds.CompanyId, sendTo, cwTicket, lastNote, s.config.MaxMsgLength)
		//if err := s.webexClient.SendMessage(ctx, msg); err != nil {
		//	return fmt.Errorf("sending webex message: %w", err)
		//}
	}

	return nil
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
