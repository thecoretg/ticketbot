// Package msgtemplate renders notification text from a small {{placeholder}} template language.
// It is shared by the workflow validator (to reject unknown placeholders on save) and the notifier
// (to render the message per recipient).
package msgtemplate

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Placeholder documents one token an admin can use in a notify message.
type Placeholder struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Placeholders is every token Render understands, in the order the editor shows them.
var Placeholders = []Placeholder{
	{"event", "\"New Ticket\" or \"Ticket Updated\""},
	{"ticket.id", "Ticket number"},
	{"ticket.link", "Ticket number as a markdown link to ConnectWise"},
	{"ticket.url", "Plain ConnectWise URL for the ticket"},
	{"ticket.summary", "Ticket summary"},
	{"board", "Board name"},
	{"status", "Current status name"},
	{"priority", "Current priority name"},
	{"company", "Company name"},
	{"contact", "Ticket contact full name"},
	{"owner", "Owner full name"},
	{"resources", "Resource full names, comma separated"},
	{"rule", "Title of the step that sent this"},
	{"note.author", "Who wrote the latest note"},
	{"note.text", "Latest note text, truncated to the configured max length"},
	{"note.quote", "Latest note as a markdown block quote, with author line"},
}

var tokenRe = regexp.MustCompile(`\{\{\s*([A-Za-z_.]+)\s*\}\}`)

var known = func() map[string]bool {
	m := make(map[string]bool, len(Placeholders))
	for _, p := range Placeholders {
		m[p.Name] = true
	}
	return m
}()

// Validate returns an error naming the first unknown placeholder in tpl, or nil.
func Validate(tpl string) error {
	for _, m := range tokenRe.FindAllStringSubmatch(tpl, -1) {
		if !known[m[1]] {
			return fmt.Errorf("unknown placeholder {{%s}}", m[1])
		}
	}
	return nil
}

// Context is everything Render can substitute.
type Context struct {
	Ticket     *models.FullTicket
	StepTitle  string // title of the notify node that sent this
	IsNew      bool
	CompanyID  string // ConnectWise company identifier for ticket links
	MaxNoteLen int    // 0 means no truncation
}

// Render substitutes every placeholder in tpl. Unknown placeholders render as empty strings.
func Render(tpl string, c Context) string {
	return tokenRe.ReplaceAllStringFunc(tpl, func(tok string) string {
		name := tokenRe.FindStringSubmatch(tok)[1]
		return c.value(name)
	})
}

func (c Context) value(name string) string {
	t := c.Ticket
	if t == nil {
		return ""
	}

	switch name {
	case "event":
		if c.IsNew {
			return "New Ticket"
		}
		return "Ticket Updated"
	case "ticket.id":
		return strconv.Itoa(t.Ticket.ID)
	case "ticket.link":
		return psa.MarkdownInternalTicketLink(t.Ticket.ID, c.CompanyID)
	case "ticket.url":
		return psa.InternalTicketLink(t.Ticket.ID, c.CompanyID)
	case "ticket.summary":
		return t.Ticket.Summary
	case "board":
		return t.Board.Name
	case "status":
		return t.Status.Name
	case "priority":
		if t.Ticket.PriorityName != nil {
			return *t.Ticket.PriorityName
		}
		return ""
	case "company":
		return t.Company.Name
	case "contact":
		if t.Contact != nil {
			return FullName(t.Contact.FirstName, t.Contact.LastName)
		}
		return ""
	case "owner":
		if t.Owner != nil {
			return FullName(t.Owner.FirstName, &t.Owner.LastName)
		}
		return ""
	case "resources":
		names := make([]string, 0, len(t.Resources))
		for _, m := range t.Resources {
			if m != nil {
				names = append(names, FullName(m.FirstName, &m.LastName))
			}
		}
		return strings.Join(names, ", ")
	case "rule":
		return c.StepTitle
	case "note.author":
		return NoteAuthor(t)
	case "note.text":
		return NoteText(t, c.MaxNoteLen)
	case "note.quote":
		return NoteQuote(t, c.MaxNoteLen)
	}

	return ""
}

// FullName joins a first and optional last name.
func FullName(first string, last *string) string {
	if last != nil && *last != "" {
		return strings.TrimSpace(first + " " + *last)
	}
	return first
}

// NoteAuthor names whoever wrote the latest note: a member, or an external contact.
func NoteAuthor(t *models.FullTicket) string {
	if t == nil || t.LatestNote == nil {
		return ""
	}
	if t.LatestNote.Member != nil {
		return FullName(t.LatestNote.Member.FirstName, &t.LatestNote.Member.LastName)
	}
	if t.LatestNote.Contact != nil {
		return FullName(t.LatestNote.Contact.FirstName, t.LatestNote.Contact.LastName)
	}
	return ""
}

// NoteText returns the latest note's text, truncated to maxLen bytes when maxLen > 0.
func NoteText(t *models.FullTicket, maxLen int) string {
	if t == nil || t.LatestNote == nil || t.LatestNote.Content == nil {
		return ""
	}
	s := *t.LatestNote.Content
	if maxLen > 0 && len(s) > maxLen {
		s = s[:maxLen] + "..."
	}
	return s
}

// NoteQuote renders the latest note as the notifier's default note block: an author line
// followed by the text as a markdown block quote. It is empty when there is no note.
func NoteQuote(t *models.FullTicket, maxLen int) string {
	if t == nil || t.LatestNote == nil || t.LatestNote.Content == nil {
		return ""
	}

	var sb strings.Builder
	if a := NoteAuthor(t); a != "" {
		fmt.Fprintf(&sb, "**Latest Note Sent By:** %s\n", a)
	}
	sb.WriteString(BlockQuote(NoteText(t, maxLen)))
	return sb.String()
}

// BlockQuote prefixes every line with "> ".
func BlockQuote(text string) string {
	parts := strings.Split(text, "\n")
	for i, p := range parts {
		parts[i] = "> " + p
	}
	return strings.Join(parts, "\n")
}
