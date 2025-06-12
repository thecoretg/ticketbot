package ticketbot

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"log/slog"
	"net/http"
	"strings"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/util"
	"tctg-automation/pkg/webex"
)

const (
	maxNoteLength = 300
)

func (s *server) handleNewTicket(c *gin.Context, ticket *connectwise.Ticket, notes []connectwise.ServiceTicketNoteAll, bs *boardSetting) {
	if bs.WebexRoomID == "" {
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("board %d is enabled, but has no specified webex room ID for new tickets", ticket.Board.ID)})
		return
	}

	log.Printf("webex room id: %s\n", bs.WebexRoomID)

	m := buildNewTicketMessage(ticket, notes)
	w := webex.NewMessageToRoom(bs.WebexRoomID, m)
	if err := s.webexClient.SendMessage(c.Request.Context(), w); err != nil {
		slog.Error("sending new ticket message", "boardName", bs.BoardName, "webexRoomId", bs.WebexRoomID, "ticketId", ticket.ID, "ticketSummary", ticket.Summary, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, fmt.Sprintf("sending message to webex room: %v", err))
		return
	}
	slog.Info("successfully sent new ticket message", "boardName", bs.BoardName, "webexRoomId", bs.WebexRoomID, "ticketId", ticket.ID, "ticketSummary", ticket.Summary)
	c.Status(http.StatusNoContent)
}

func (s *server) handleUpdatedTicket(c *gin.Context, ticket *connectwise.Ticket, notes []connectwise.ServiceTicketNoteAll, bs *boardSetting, w *connectwise.WebhookPayload) {
	var (
		mutedUsers, emails []string
		err                error
	)

	if updatedBy, present := hasUpdatedBy(w.Entity); present {
		slog.Debug("found updatedBy in entity - muting from webex messages", "updatedBy", updatedBy)
		mutedUsers = append(mutedUsers, updatedBy)

		u, err := getUserByCwID(s.db, updatedBy)
		if err != nil {
			slog.Error("failed to get user by ConnectWise ID", "id", updatedBy, "error", err)
			util.ErrorJSON(c, http.StatusInternalServerError, fmt.Sprintf("getting user by ConnectWise ID: %v", err))
			return
		}

		if u != nil && u.IgnoreUpdate {
			slog.Debug("user is marked to ignore updates, not sending message", "id", updatedBy)
			c.JSON(http.StatusNoContent, gin.H{"message": fmt.Sprintf("user %s is marked to ignore updates, not sending message", updatedBy)})
			return
		}
	}

	latestNote := mostRecentNote(notes)
	if latestNote != nil {
		mutedUsers = append(mutedUsers, noteSenderName(latestNote))
	}

	if ticket.Resources != "" {
		slog.Debug("ticket has resources, fetching emails", "ticketId", ticket.ID, "resources", ticket.Resources)
		emails, err = s.getAndStoreResourceEmails(c.Request.Context(), ticket.Resources, mutedUsers)
		if err != nil {
			util.ErrorJSON(c, http.StatusInternalServerError, fmt.Sprintf("getting emails from resource ids: %v", err))
			return
		}
	}

	if len(emails) == 0 {
		slog.Debug("no recipients found for updated ticket message", "ticketId", ticket.ID, "boardName", bs.BoardName)
		c.JSON(http.StatusNoContent, gin.H{"message": "no resources to send messages to"})
		return
	}

	m := buildUpdatedTicketMessage(ticket, notes)
	for _, r := range emails {
		w := webex.NewMessageToPerson(r, m)
		if err := s.webexClient.SendMessage(c.Request.Context(), w); err != nil {
			util.ErrorJSON(c, http.StatusInternalServerError, fmt.Sprintf("sending message to user %s: %v", r, err))
			return
		}
		slog.Info("successfully updated new ticket message", "boardName", bs.BoardName, "email", r, "ticketId", ticket.ID, "ticketSummary", ticket.Summary)
	}
	c.Status(http.StatusNoContent)
}

func isMuted(email string, mutedUsers []string) bool {
	for _, e := range mutedUsers {
		if e == email {
			return true
		}
	}
	return false
}

func hasUpdatedBy(entity string) (string, bool) {
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(entity), &m); err != nil {
		return "", false
	}

	info, ok := m["_info"].(map[string]interface{})
	if !ok {
		return "", false
	}

	val, present := info["updatedBy"]
	if !present {
		return "", false
	}

	strVal, ok := val.(string)
	return strVal, strVal != ""
}

func (s *server) getBoard(ticket *connectwise.Ticket) (*boardSetting, error) {
	b, err := getBoardByID(s.db, ticket.Board.ID)
	if err != nil {
		return nil, fmt.Errorf("getting board by ID %d: %w", ticket.Board.ID, err)
	}

	if b == nil {
		return nil, nil
	}

	return b, nil
}

func buildNewTicketMessage(t *connectwise.Ticket, n []connectwise.ServiceTicketNoteAll) string {
	m := fmt.Sprintf("**New:** %s %s", ticketLink(t), t.Summary)

	// add requester contact and company name, or just company if no contact (rare)
	r := fmt.Sprintf("**Requester:** %s (No Contact)", t.Company.Name)
	if t.Contact.Name != "" {
		r = fmt.Sprintf("**Requester:** %s (%s)", t.Contact.Name, t.Company.Name)
	}
	m += fmt.Sprintf("\n%s", r)

	// add most recent note if present
	mr := mostRecentNote(n)
	if mr != nil {
		// trim note text and add ... if it exceeds the maximum
		noteTxt := mr.Text
		if len(noteTxt) > maxNoteLength {
			noteTxt = noteTxt[:maxNoteLength] + "..."
		}

		m += fmt.Sprintf("\n**Latest Note:** %s\n"+
			"%s",
			noteSenderName(mr),
			addBlockQuotes(noteTxt),
		)
	}

	return m
}

func buildUpdatedTicketMessage(t *connectwise.Ticket, n []connectwise.ServiceTicketNoteAll) string {
	m := fmt.Sprintf("**New Response:** %s %s", ticketLink(t), t.Summary)

	// add most recent note if present
	mr := mostRecentNote(n)
	if mr != nil {
		// trim note text and add ... if it exceeds the maximum
		noteTxt := mr.Text
		if len(noteTxt) > maxNoteLength {
			noteTxt = noteTxt[:maxNoteLength] + "..."
		}

		m += fmt.Sprintf("\n**Latest Note:** %s\n"+
			"%s",
			noteSenderName(mr),
			addBlockQuotes(noteTxt),
		)
	}

	return m
}

func noteSenderName(n *connectwise.ServiceTicketNoteAll) string {
	if n.Member.Name != "" {
		return n.Member.Name
	} else if n.Contact.Name != "" {
		return n.Contact.Name
	} else {
		return "N/A"
	}
}

func mostRecentNote(n []connectwise.ServiceTicketNoteAll) *connectwise.ServiceTicketNoteAll {
	for _, note := range n {
		if note.Text != "" {
			return &note
		}
	}

	return nil
}

func ticketLink(t *connectwise.Ticket) string {
	return fmt.Sprintf("[%d](https://na.myconnectwise.net/v4_6_release/services/system_io/Service/fv_sr100_request.rails?service_recid=%d&companyName=securenetit)", t.ID, t.ID)
}

func addBlockQuotes(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line == "" {
			lines[i] = ">"
		} else {
			lines[i] = "> " + line
		}
	}

	return strings.Join(lines, "\n")
}
