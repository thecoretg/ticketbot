package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type setNotifyPayload struct {
	Enabled bool `json:"enabled"`
}

func (cl *Client) handleSetAttemptNotify(c *gin.Context) {
	p := &setNotifyPayload{}
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := cl.setAttemptNotify(c.Request.Context(), p.Enabled); err != nil {
		c.Error(fmt.Errorf("setting notify state: %w", err))
		return
	}

	c.Status(http.StatusOK)
}

func (cl *Client) handleListWebexRooms(c *gin.Context) {
	// TODO: query params?
	rooms, err := cl.WebexClient.ListRooms(nil)
	if err != nil {
		c.Error(fmt.Errorf("listing rooms: %w", err))
		return
	}

	if rooms == nil {
		rooms = []webex.Room{}
	}

	c.JSON(http.StatusOK, rooms)
}

func (cl *Client) makeAndSendWebexMsgs(ctx context.Context, action string, cd *cwData, sd *storedData) error {

	messages, err := cl.makeWebexMsgs(ctx, action, cd, sd)
	if err != nil {
		return fmt.Errorf("creating webex messages: %w", err)
	}

	if messages == nil {
		slog.Debug("no messages to send", "ticket_id", sd.ticket.ID, "note_id", sd.note.ID)
		return nil
	}

	slog.Debug("created webex messages", "action", action, "ticket_id", sd.ticket.ID, "board_name", sd.board.Name, "total_messages", len(messages))
	for _, msg := range messages {
		_, err := cl.WebexClient.PostMessage(&msg)
		if err != nil {
			// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
			// the person on the ticket doesn't have an account, or the same email address, in Webex.
			slog.Warn("error sending webex message", "action", action, "ticket_id", sd.ticket.ID, "room_id", msg.RoomId, "person", msg.ToPersonEmail, "error", err)
		}

		sentTo := "webex room"
		if msg.ToPersonEmail != "" {
			sentTo = msg.ToPersonEmail
		}

		slog.Debug("notification sent", "action", action, "ticket_id", sd.ticket.ID, "board_name", sd.board.Name, "sent_to", sentTo)
	}

	return nil
}

// makeWebexMsgs constructs a message - it handles new tickets and updated tickets, and determines which Webex room, or which people,
// the message should be sent to.
func (cl *Client) makeWebexMsgs(ctx context.Context, action string, cd *cwData, sd *storedData) ([]webex.Message, error) {
	var body string
	body += cl.messageHeader(action, cd)

	// add company name if present (even Catchall is considered a company, so this will always exist)
	if cd.ticket.Company.Name != "" {
		body += fmt.Sprintf("\n**Company:** %s", cd.ticket.Company.Name)
	}

	// add ticket contact name if exists (not always true)
	if cd.ticket.Contact.Name != "" {
		body += fmt.Sprintf("\n**Ticket Contact:** %s", cd.ticket.Contact.Name)
	}

	if cd.note.Text != "" {
		body += cl.messageText(cd)
	}

	// Divider line for easily distinguishable breaks in notifications
	body += fmt.Sprintf("\n\n---")

	var messages []webex.Message
	if action == "added" {
		slog.Debug("creating message for new ticket", "ticket_id", sd.ticket.ID, "board_name", sd.board.Name, "rooms_to_notify", roomNames(sd.notifyRooms))
		for _, r := range sd.notifyRooms {
			messages = append(messages, webex.NewMessageToRoom(r.WebexID, body))
		}
	} else if action == "updated" {
		sendTo, err := cl.getSendTo(ctx, sd)
		if len(sendTo) > 0 {
			slog.Debug("got send-to list", "ticket_id", sd.ticket.ID, "note_id", sd.note.ID, "send_to", sendTo)
		} else {
			slog.Debug("send-to list is empty", "ticket_id", sd.ticket.ID, "note_id", sd.note.ID)
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

func roomNames(rooms []db.WebexRoom) []string {
	var names []string
	for _, r := range rooms {
		names = append(names, r.Name)
	}

	return names
}

// getSendTo creates a list of emails to send notifications to, factoring in who made the most
// recent update and any other exclusions passed in by the Config.
func (cl *Client) getSendTo(ctx context.Context, sd *storedData) ([]string, error) {
	var (
		excludedMembers []int
		sendToMembers   []db.CwMember
	)

	// if the sender of the note is a member, exclude them from messages since they don't need a notification for their own note
	if sd.note.MemberID != nil {
		excludedMembers = append(excludedMembers, *sd.note.MemberID)
	}

	// if there are multiple resources in the string, there are spaces after commas - trim those out
	resources := strings.Split(*sd.ticket.Resources, ",")
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

	var emails []string
	for _, m := range sendToMembers {
		if m.PrimaryEmail != "" {
			emails = append(emails, m.PrimaryEmail)
		}
	}

	return emails, nil
}

func (cl *Client) messageHeader(action string, cd *cwData) string {
	var header string
	if action == "added" {
		header += "**New Ticket:** "
	} else {
		header += "**Ticket Updated:** "
	}

	// add clickable ticket ID with link to ticket, with ticket title
	header += fmt.Sprintf("%s %s", psa.MarkdownInternalTicketLink(cd.ticket.ID, cl.Config.CWCompanyID), cd.ticket.Summary)
	return header
}

func (cl *Client) messageText(cd *cwData) string {
	var body string
	sender := getSenderName(cd)
	if sender != nil {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", *sender)
	}

	text := cd.note.Text
	if len(text) > cl.Config.MaxMsgLength {
		text = text[:cl.Config.MaxMsgLength] + "..."
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
func getSenderName(cd *cwData) *string {
	if cd.note.Member.Name != "" {
		return &cd.note.Member.Name
	} else if cd.note.CreatedBy != "" {
		return &cd.note.CreatedBy
	} else if cd.note.Contact.Name != "" {
		return &cd.note.Contact.Name
	}

	return nil
}
