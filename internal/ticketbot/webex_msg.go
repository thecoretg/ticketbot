package ticketbot

import (
	"fmt"
	"slices"
	"strings"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"
)

// makeWebexMsgs constructs a message - it handles new tickets and updated tickets, and determines which Webex room, or which people,
// the message should be sent to.
func (s *server) makeWebexMsgs(action, updatedBy string, board *Board, ticket *connectwise.Ticket, note *connectwise.ServiceTicketNote) ([]webex.MessagePostBody, error) {
	var body string
	body += s.messageHeader(action, ticket)

	// add company name if present (even Catchall is considered a company, so this will always exist)
	if ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", ticket.Company.Name)
	}

	// add ticket contact name if exists (not always true)
	if ticket.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", ticket.Contact.Name)
	}

	if note.Text != "" {
		body += s.messageText(note)
	}

	var messages []webex.MessagePostBody
	if action == "added" {
		messages = append(messages, webex.NewMessageToRoom(board.WebexRoomID, body))
	} else if action == "updated" {
		sendTo, err := s.getSendTo(updatedBy, ticket)
		if err != nil {
			return nil, fmt.Errorf("getting users to send to: %w", err)
		}

		for _, email := range sendTo {
			messages = append(messages, webex.NewMessageToPerson(email, body))
		}
	}

	return messages, nil
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
	if identifiers == "" {
		return nil, nil
	}

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

func (s *server) messageHeader(action string, ticket *connectwise.Ticket) string {
	var header string
	if action == "added" {
		header += "**New Ticket:** "
	} else {
		header += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	header += fmt.Sprintf("%s %s", connectwise.MarkdownInternalTicketLink(ticket.ID, s.cwCompanyID), ticket.Summary)
	return header
}

func (s *server) messageText(note *connectwise.ServiceTicketNote) string {
	var body string
	sender := getSenderName(note)
	if sender != nil {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", *sender)
	}

	text := note.Text
	if len(text) > s.config.MaxMsgLength {
		text = text[:s.config.MaxMsgLength] + "..."
	}
	body += fmt.Sprintf("\n%s", blockQuoteText(text))
	return body
}

// getSenderName determines the name of the sender of a note. It checks for members in Connectwise and external contacts from companies.
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
