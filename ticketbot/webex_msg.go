package ticketbot

import (
	"context"
	"fmt"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/webex"
	"log/slog"
	"slices"
	"strings"
)

func (s *Server) makeAndSendWebexMsgs(ctx context.Context, action string, cwData *cwData, storedData *storedData) error {
	messages, err := s.makeWebexMsgs(action, cwData, storedData)
	if err != nil {
		return fmt.Errorf("creating webex messages: %w", err)
	}

	for _, msg := range messages {
		if err := s.webexClient.PostMessage(ctx, msg); err != nil {
			// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
			// the person on the ticket doesn't have an account, or the same email address, in Webex.
			slog.Warn("error sending webex message", "action", action, "ticket_id", storedData.ticket.ID, "room_id", msg.RoomId, "person", msg.Person, "error", err)
		}

		sentTo := "webex room"
		if msg.Person != "" {
			sentTo = msg.Person
		}

		slog.Info("notification sent", "action", action, "ticket_id", storedData.ticket.ID, "board_name", storedData.board.Name, "sent_to", sentTo)
	}

	return nil
}

// makeWebexMsgs constructs a message - it handles new tickets and updated tickets, and determines which Webex room, or which people,
// the message should be sent to.
func (s *Server) makeWebexMsgs(action string, cwData *cwData, storedData *storedData) ([]webex.Message, error) {
	var body string
	body += s.messageHeader(action, cwData)

	// add company name if present (even Catchall is considered a company, so this will always exist)
	if cwData.ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", cwData.ticket.Company.Name)
	}

	// add ticket contact name if exists (not always true)
	if cwData.ticket.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", cwData.ticket.Contact.Name)
	}

	if cwData.note.Text != "" {
		body += s.messageText(cwData)
	}

	// Divider line for easily distinguishable breaks in notifications
	body += fmt.Sprintf("\n\n---")

	var messages []webex.Message
	if action == "added" {
		messages = append(messages, webex.NewMessageToRoom(storedData.board.WebexRoomID, body))
	} else if action == "updated" {
		sendTo, err := s.getSendTo(storedData)
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
func (s *Server) getSendTo(storedData *storedData) ([]string, error) {
	var excludedMembers []string
	for _, m := range s.config.ExcludedCWMembers {
		excludedMembers = append(excludedMembers, m)
	}

	if storedData.ticket.UpdatedBy != nil {
		excludedMembers = append(excludedMembers, storedData.ticket.UpdatedBy)
	}

	identifiers := filterOutExcluded(excludedMembers, storedData.ticket.Resources)
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

func (s *Server) messageHeader(action string, cwData *cwData) string {
	var header string
	if action == "added" {
		header += "**New Ticket:** "
	} else {
		header += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	header += fmt.Sprintf("%s %s", connectwise.MarkdownInternalTicketLink(cwData.ticket.ID, s.cwCompanyID), cwData.ticket.Summary)
	return header
}

func (s *Server) messageText(cwData *cwData) string {
	var body string
	sender := getSenderName(cwData)
	if sender != nil {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", *sender)
	}

	text := cwData.note.Text
	if len(text) > s.config.MaxMsgLength {
		text = text[:s.config.MaxMsgLength] + "..."
	}
	body += fmt.Sprintf("\n%s", blockQuoteText(text))
	return body
}

// getSenderName determines the name of the sender of a note. It checks for members in Connectwise and external contacts from companies.
func getSenderName(cwData *cwData) *string {
	if cwData.note.Member.Name != "" {
		return &cwData.note.Member.Name
	} else if cwData.note.CreatedBy != "" {
		return &cwData.note.CreatedBy
	} else if cwData.note.Contact.Name != "" {
		return &cwData.note.Contact.Name
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
