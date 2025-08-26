package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
	"github.com/thecoretg/ticketbot/webex"
)

func (s *Server) makeAndSendWebexMsgs(ctx context.Context, action string, cwData *cwData, storedData *storedData) error {
	messages, err := s.makeWebexMsgs(ctx, action, cwData, storedData)
	if err != nil {
		return fmt.Errorf("creating webex messages: %w", err)
	}

	if messages == nil {
		slog.Debug("no messages to send", "ticket_id", storedData.ticket.ID, "note_id", storedData.note.ID)
		return nil
	}

	slog.Debug("created webex messages", "action", action, "ticket_id", storedData.ticket.ID, "board_name", storedData.board.Name, "total_messages", len(messages))
	for _, msg := range messages {
		_, err := s.WebexClient.PostMessage(&msg)
		if err != nil {
			// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
			// the person on the ticket doesn't have an account, or the same email address, in Webex.
			slog.Warn("error sending webex message", "action", action, "ticket_id", storedData.ticket.ID, "room_id", msg.RoomId, "person", msg.ToPersonEmail, "error", err)
		}

		sentTo := "webex room"
		if msg.ToPersonEmail != "" {
			sentTo = msg.ToPersonEmail
		}

		slog.Info("notification sent", "action", action, "ticket_id", storedData.ticket.ID, "board_name", storedData.board.Name, "sent_to", sentTo)
	}

	return nil
}

// makeWebexMsgs constructs a message - it handles new tickets and updated tickets, and determines which Webex room, or which people,
// the message should be sent to.
func (s *Server) makeWebexMsgs(ctx context.Context, action string, cwData *cwData, storedData *storedData) ([]webex.Message, error) {
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
		slog.Debug("creating message for new ticket", "ticket_id", storedData.ticket.ID, "board_name", storedData.board.Name, "webex_room_id", storedData.board.WebexRoomID)
		messages = append(messages, webex.NewMessageToRoom(*storedData.board.WebexRoomID, body))
	} else if action == "updated" {
		sendTo, err := s.getSendTo(ctx, storedData)
		if len(sendTo) > 0 {
			slog.Debug("got send-to list", "ticket_id", storedData.ticket.ID, "note_id", storedData.note.ID, "send_to", sendTo)
		} else {
			slog.Debug("send-to list is empty", "ticket_id", storedData.ticket.ID, "note_id", storedData.note.ID)
			return nil, nil
		}

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
// recent update and any other exclusions passed in by the Config.
func (s *Server) getSendTo(ctx context.Context, storedData *storedData) ([]string, error) {
	var (
		excludedMembers []int
		sendToMembers   []db.CwMember
	)

	// if the sender of the note is a member, exclude them from messages since they don't need a notification for their own note
	if storedData.note.MemberID != nil {
		excludedMembers = append(excludedMembers, *storedData.note.MemberID)
	}

	resources := strings.Split(*storedData.ticket.Resources, ",")
	for _, r := range resources {
		m, err := s.Queries.GetMemberByIdentifier(ctx, r)
		if err != nil {
			slog.Warn("getSendTo: couldn't get member data", "resource", r, "error", err)
			continue
		}

		if m.ID != 0 && !intSliceContains(excludedMembers, m.ID) {
			sendToMembers = append(sendToMembers, m)
		}
	}

	var emails []string
	for _, m := range sendToMembers {
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
	if len(text) > s.Config.MaxMsgLength {
		text = text[:s.Config.MaxMsgLength] + "..."
	}
	body += fmt.Sprintf("\n%s", blockQuoteText(text))
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
