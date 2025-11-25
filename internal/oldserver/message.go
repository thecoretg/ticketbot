package oldserver

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/external/webex"
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
		rs.noNotiReason = "NONE_TO_SEND"
		return rs, nil
	}

	var msgErr error
	for _, m := range rs.messagesToSend {
		if _, err := cl.MessageSender.PostMessage(&m); err != nil {
			// Don't fully exit, just warn, if a message isn't sent. Sometimes, this will happen if
			// the person on the ticket doesn't have an account, or the same email address, in Webex.
			// We will also set msgErr so that when the loop is done, it will exit as an error
			rs.logger.Warn("error sending webex message",
				slog.String("type", m.RecipientType),
				slog.String("name", m.RecipientName),
				slog.String("error", err.Error()))
			rs.failedNotis = append(rs.failedNotis, m.RecipientName)
			msgErr = fmt.Errorf("sending message to %s %s: %w", m.RecipientType, m.RecipientName, err)
		} else {
			rs.logger.Debug("success sending webex message",
				slog.String("type", m.RecipientType),
				slog.String("name", m.RecipientName))
			rs.successNotis = append(rs.successNotis, m.RecipientName)
		}
	}

	return rs, msgErr
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
			rs.logger.Debug("adding room to send list", slog.String("name", r.Name))
			rs.roomsToNotify = append(rs.roomsToNotify, r)
			messages = append(messages, webex.NewMessageToRoom(r.WebexID, r.Name, body))
		}
	} else if rs.action == "updated" {
		rs = cl.getSendTo(ctx, rs)
		if len(rs.membersToNotify) == 0 {
			return rs, nil
		}

		for _, m := range rs.membersToNotify {
			rs.logger.Debug("adding person to send list", slog.String("email", m.PrimaryEmail))
			messages = append(messages, webex.NewMessageToPerson(m.PrimaryEmail, body))
		}
	}

	rs.messagesToSend = messages
	return rs, nil
}

func forwardIsActive(fwd WebexUserForward) bool {
	active := false
	if fwd.Enabled {
		active = dateRangeActive(fwd.StartDate, fwd.EndDate)
	}

	return active
}

func dateRangeActive(start, end *time.Time) bool {
	now := time.Now()
	if start == nil {
		return false
	}

	if end == nil {
		return now.After(*start)
	}

	return now.After(*start) && now.Before(*end)
}

// getSendTo creates a list of emails to send notifications to, factoring in who made the most
// recent update and any other exclusions passed in by the cfgOld.
func (cl *Client) getSendTo(ctx context.Context, rs *requestState) *requestState {
	var excludedMembers []int

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

	if rs.membersToNotify == nil {
		rs.membersToNotify = []db.CwMember{}
	}

	for _, r := range resources {
		m, err := cl.Queries.GetMemberByIdentifier(ctx, r)
		if err != nil {
			rs.logger.Warn("getSendTo: couldn't get member data, skipping for notifications", "resource", r, "error", err)
			continue
		}

		if m.ID != 0 && !intSliceContains(excludedMembers, m.ID) {
			rs.membersToNotify = append(rs.membersToNotify, m)
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
	if sender != "" {
		body += fmt.Sprintf("\n**Latest Note Sent By:** %s", sender)
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
func getSenderName(cd *connectwiseData) string {
	if cd.note.Member.Name != "" {
		return cd.note.Member.Name
	} else if cd.note.CreatedBy != "" {
		return cd.note.CreatedBy
	} else if cd.note.Contact.Name != "" {
		return cd.note.Contact.Name
	}

	return ""
}
