package notifier

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type Request struct {
	Ticket          *models.FullTicket
	Notifications   []models.TicketNotification
	MessagesToSend  []Message
	MessagesSent    []Message
	MessagesErrored []Message
	NoNotiReason    string
	Error           error
}

type Message struct {
	MsgType      string
	WebexMsg     webex.Message
	ToEmail      *string
	WebexRoom    *models.WebexRoom
	Notification models.TicketNotification
	SendError    error
}

func newRequest(ticket *models.FullTicket) *Request {
	return &Request{
		Ticket:         ticket,
		Notifications:  []models.TicketNotification{},
		MessagesToSend: []Message{},
		NoNotiReason:   "",
		Error:          nil,
	}
}

func newMessage(msgType string, wm webex.Message, toEmail *string, wr *models.WebexRoom, n models.TicketNotification) Message {
	return Message{
		MsgType:      msgType,
		WebexMsg:     wm,
		ToEmail:      toEmail,
		WebexRoom:    wr,
		Notification: n,
	}
}

func (s *Service) ProcessTicket(ctx context.Context, ticket *models.FullTicket, isNew bool) *Request {
	req := newRequest(ticket)

	notifiers, err := s.Notifiers.ListByBoard(ctx, ticket.Board.ID)
	if err != nil {
		req.Error = fmt.Errorf("listing notifiers for board: %w", err)
		return req
	}

	if len(notifiers) == 0 {
		req.NoNotiReason = "no notifiers found"
		return req
	}

	if isNew {
		var rooms []models.WebexRoom
		for _, n := range notifiers {
			if !n.NotifyEnabled {
				continue
			}

			r, err := s.Rooms.Get(ctx, n.WebexRoomID)
			if err != nil {
				req.Error = fmt.Errorf("getting webex room from notifier: %w", err)
				return req
			}

			rooms = append(rooms, r)
		}
		req.MessagesToSend = s.makeNewTicketMessages(rooms, ticket)

	} else {

		if ticket.LatestNote == nil {
			req.NoNotiReason = "no note found for ticket"
			return req
		}

		exists, err := s.checkExistingNoti(ctx, ticket.LatestNote.ID)
		if err != nil {
			req.Error = fmt.Errorf("checking for existing notification for ticket note: %w", err)
			return req
		}

		if exists {
			req.NoNotiReason = "note already notified"
			return req
		}

		emails := s.getUpdateRecipientEmails(ctx, ticket)
		if len(emails) == 0 {
			req.NoNotiReason = "no resources to notify"
			return req
		}
		req.MessagesToSend = s.makeUpdatedTicketMessages(ticket, emails)
	}

	if len(req.MessagesToSend) > 0 {
		for _, m := range req.MessagesToSend {
			msg := s.sendNotification(ctx, &m)
			if msg.SendError != nil {
				req.MessagesErrored = append(req.MessagesErrored, *msg)
				continue
			}

			req.MessagesSent = append(req.MessagesSent, *msg)
		}
	}

	return req
}

func (s *Service) checkExistingNoti(ctx context.Context, noteID int) (bool, error) {
	exists, err := s.Notifications.ExistsForNote(ctx, noteID)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}

	return false, nil
}

func (s *Service) makeNewTicketMessages(rooms []models.WebexRoom, ticket *models.FullTicket) []Message {
	header := "**New Ticket:** "
	body := makeMessageBody(ticket, header, s.MaxMessageLength)

	var msgs []Message
	for _, r := range rooms {
		wm := webex.NewMessageToRoom(r.WebexID, r.Name, body)

		n := &models.TicketNotification{
			TicketID:    ticket.Ticket.ID,
			WebexRoomID: &r.ID,
			Sent:        true,
		}

		if ticket.LatestNote != nil {
			n.TicketNoteID = &ticket.LatestNote.ID
		}

		msgs = append(msgs, newMessage("new_ticket", wm, nil, &r, *n))
	}

	return msgs
}

func (s *Service) makeUpdatedTicketMessages(ticket *models.FullTicket, emails []string) []Message {
	header := "**Ticket Updated:** "
	body := makeMessageBody(ticket, header, s.MaxMessageLength)

	var msgs []Message
	for _, e := range emails {
		wm := webex.NewMessageToPerson(e, body)

		n := &models.TicketNotification{
			TicketID:    ticket.Ticket.ID,
			SentToEmail: &e,
			Sent:        true,
		}

		if ticket.LatestNote != nil {
			n.TicketNoteID = &ticket.LatestNote.ID
		}

		msgs = append(msgs, newMessage("updated_ticket", wm, &e, nil, *n))
	}

	return msgs
}

func (s *Service) getUpdateRecipientEmails(ctx context.Context, ticket *models.FullTicket) []string {
	var excluded []models.Member

	// if the sender of the note is a member, exclude them from messages;
	// they don't need a notification for their own note
	if ticket.LatestNote != nil && ticket.LatestNote.Member != nil {
		excluded = append(excluded, *ticket.LatestNote.Member)
	}

	var emails []string
	for _, r := range ticket.Resources {
		if memberSliceContains(excluded, r) {
			continue
		}

		e, err := s.forwardsToEmails(ctx, r.PrimaryEmail)
		if err != nil {
			slog.Warn("notifier: checking forwards for email", "email", r.PrimaryEmail, "error", err)
		}

		emails = append(emails, e...)
	}

	return filterDuplicateEmails(emails)
}

func (s *Service) forwardsToEmails(ctx context.Context, email string) ([]string, error) {
	noFwdSlice := []string{email}
	fwds, err := s.Forwards.ListByEmail(ctx, email)
	if err != nil {
		return noFwdSlice, fmt.Errorf("checking forwards: %w", err)
	}

	if len(fwds) == 0 {
		return noFwdSlice, nil
	}

	activeFwds := filterActiveFwds(fwds)
	if len(activeFwds) == 0 {
		return noFwdSlice, nil
	}

	var emails []string
	for _, f := range activeFwds {
		if f.UserKeepsCopy {
			emails = append(emails, email)
			break
		}
	}

	for _, f := range activeFwds {
		emails = append(emails, f.DestEmail)
	}

	return emails, nil
}

func (s *Service) sendNotification(ctx context.Context, m *Message) *Message {
	_, err := s.MessageSender.PostMessage(&m.WebexMsg)
	if err != nil {
		m.SendError = fmt.Errorf("sending webex message: %w", err)
	}

	m.Notification, err = s.Notifications.Insert(ctx, m.Notification)
	if err != nil {
		m.SendError = fmt.Errorf("message was sent, but error inserting record: %w", err)
	}

	return m
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
	body += fmt.Sprintf("\n\n---")
	return body
}

func filterDuplicateEmails(emails []string) []string {
	seenEmails := make(map[string]struct{})
	for _, e := range emails {
		if _, ok := seenEmails[e]; !ok {
			seenEmails[e] = struct{}{}
		}
	}

	var uniqueEmails []string
	for e := range seenEmails {
		uniqueEmails = append(uniqueEmails, e)
	}

	return uniqueEmails
}

func filterActiveFwds(fwds []models.UserForward) []models.UserForward {
	var activeFwds []models.UserForward
	for _, f := range fwds {
		if f.Enabled && dateRangeActive(f.StartDate, f.EndDate) {
			activeFwds = append(activeFwds, f)
		}
	}

	return activeFwds
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

func memberSliceContains(members []models.Member, check models.Member) bool {
	for _, x := range members {
		if x.ID == check.ID {
			return true
		}
	}

	return false
}
