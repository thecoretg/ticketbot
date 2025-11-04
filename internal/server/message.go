package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type messageSender interface {
	PostMessage(message *webex.Message) (*webex.Message, error)
	ListRooms(params map[string]string) ([]webex.Room, error)
}

func (cl *Client) makeAndSendMessages(ctx context.Context, rs *requestState) (*requestState, error) {
	var err error
	rs, err = cl.makeMessages(ctx, rs)
	if err != nil {
		return rs, fmt.Errorf("creating webex messages: %w", err)
	}

	if rs.messagesToSend == nil || len(rs.messagesToSend) == 0 {
		return rs, nil
	}

	for _, msg := range rs.messagesToSend {
		_, err := cl.MessageSender.PostMessage(&msg)
		if err != nil {
			// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
			// the person on the ticket doesn't have an account, or the same email address, in Webex.
			slog.Warn("error sending webex message", "action", rs.action, "ticket_id", rs.dbData.ticket.ID, "room_id", msg.RoomId, "person", msg.ToPersonEmail, "error", err)
		}
	}

	return rs, nil
}

// makeMessages constructs a message - it handles new tickets and updated tickets, and determines which Webex room, or which people,
// the message should be sent to.
func (cl *Client) makeMessages(ctx context.Context, rs *requestState) (*requestState, error) {
	var body string
	body += cl.messageHeader(rs.action, rs.cwData)

	// add company name if present (even Catchall is considered a company, so this will always exist)
	if rs.dbData.company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", rs.dbData.company.Name)
	}

	// add ticket contact name if exists (not always true)
	if rs.cwData.ticket.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", rs.cwData.ticket.Contact.Name)
	}

	if rs.cwData.note.Text != "" {
		body += cl.messageText(rs.cwData)
	}

	// Divider line for easily distinguishable breaks in notifications
	body += fmt.Sprintf("\n\n---")

	var messages []webex.Message
	if rs.action == "added" {
		for _, r := range rs.dbData.enabledRooms {
			rs.roomsNotify = append(rs.roomsNotify, r.Name)
			messages = append(messages, webex.NewMessageToRoom(r.WebexID, body))
		}
	} else if rs.action == "updated" {
		rs = cl.getSendTo(ctx, rs)
		if len(rs.peopleNotify) == 0 {
			return rs, nil
		}

		for _, email := range rs.peopleNotify {
			rs.peopleNotify = append(rs.peopleNotify, email)
			messages = append(messages, webex.NewMessageToPerson(email, body))
		}
	}

	rs.messagesToSend = messages
	return rs, nil
}

// getSendTo creates a list of emails to send notifications to, factoring in who made the most
// recent update and any other exclusions passed in by the cfgOld.
func (cl *Client) getSendTo(ctx context.Context, rs *requestState) *requestState {
	var (
		excludedMembers []int
		sendToMembers   []db.CwMember
	)

	// if the sender of the note is a member, exclude them from messages since they don't need a notification for their own note
	if rs.dbData.note.MemberID != nil {
		excludedMembers = append(excludedMembers, *rs.dbData.note.MemberID)
	}

	// if there are multiple resources in the string, there are spaces after commas - trim those out
	var resources []string
	if rs.dbData.ticket.Resources != nil {
		resources = strings.Split(*rs.dbData.ticket.Resources, ",")
	}

	for i, r := range resources {
		resources[i] = strings.TrimSpace(r)
	}

	for _, r := range resources {
		m, err := cl.Queries.GetMemberByIdentifier(ctx, r)
		if err != nil {
			slog.Warn("getSendTo: couldn't get member data", "resource", r, "error", err)
			continue
		}

		if m.ID != 0 && !intSliceContains(excludedMembers, m.ID) {
			sendToMembers = append(sendToMembers, m)
		}
	}

	if rs.peopleNotify == nil {
		rs.peopleNotify = []string{}
	}

	for _, m := range sendToMembers {
		if m.PrimaryEmail != "" {
			rs.peopleNotify = append(rs.peopleNotify, m.PrimaryEmail)
		}
	}

	return rs
}

func (cl *Client) messageHeader(action string, cd *connectwiseData) string {
	var header string
	if action == "added" {
		header += "**New Ticket:** "
	} else {
		header += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	header += fmt.Sprintf("%s %s", psa.MarkdownInternalTicketLink(cd.ticket.ID, cl.Creds.CWCompanyID), cd.ticket.Summary)
	return header
}

func (cl *Client) messageText(cd *connectwiseData) string {
	var body string
	sender := getSenderName(cd)
	if sender != nil {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", *sender)
	}

	text := cd.note.Text
	if len(text) > cl.Config.MaxMessageLength {
		text = text[:cl.Config.MaxMessageLength] + "..."
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
func getSenderName(cd *connectwiseData) *string {
	if cd.note.Member.Name != "" {
		return &cd.note.Member.Name
	} else if cd.note.CreatedBy != "" {
		return &cd.note.CreatedBy
	} else if cd.note.Contact.Name != "" {
		return &cd.note.Contact.Name
	}

	return nil
}
