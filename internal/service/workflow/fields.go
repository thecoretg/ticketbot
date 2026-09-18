package workflow

import (
	"strings"

	"github.com/thecoretg/ticketbot/models"
)

// ConditionField documents one path the visual condition builder offers. Any field on the
// ConnectWise ticket JSON works in a hand-written condition; this list is what the builder knows
// how to render, and how the advanced editor's text is mapped back onto builder rows.
type ConditionField struct {
	Path  string `json:"path"`
	Label string `json:"label"`
	Group string `json:"group"`
	// Type drives the operator list and value widget:
	//   string      free text
	//   number      free numeric
	//   bool        is true / is false
	//   ref         an id chosen from Source (compiles to `path = 123` / `path in (…)`)
	//   identifier  a member identifier chosen from Source (compiles to `path = 'jdoe'`)
	//   list        a comma-separated string of member identifiers; `contains` against Source
	Type string `json:"type"`
	// Source names the lookup that supplies values: statuses, priorities, members, boards,
	// companies, contacts. Empty for free-form fields.
	Source      string `json:"source,omitempty"`
	Description string `json:"description,omitempty"`
	// ListType is the admin-list item type this field can be checked against with `in list`
	// (contact, company). Derived from Source; empty when no list type applies.
	ListType string `json:"list_type,omitempty"`
}

func init() {
	for i := range ConditionFields {
		if t, ok := models.ListItemTypeForSource(ConditionFields[i].Source); ok {
			ConditionFields[i].ListType = string(t)
		}
	}
}

// FieldByPath finds a builder field by its path, case-insensitively.
func FieldByPath(path string) (ConditionField, bool) {
	for _, f := range ConditionFields {
		if strings.EqualFold(f.Path, path) {
			return f, true
		}
	}
	return ConditionField{}, false
}

const (
	groupTicket   = "Ticket"
	groupNote     = "Latest note"
	groupChanged  = "What changed"
	groupPrevious = "Previous values"
)

// ConditionFields is served by GET /workflows/fields.
var ConditionFields = []ConditionField{
	// Ticket
	{"summary", "Summary", groupTicket, "string", "", "Ticket summary", ""},
	{"id", "Ticket number", groupTicket, "number", "", "", ""},
	{"status/id", "Status", groupTicket, "ref", "statuses", "", ""},
	{"status/name", "Status (by name)", groupTicket, "string", "", "Matches the status name as text", ""},
	{"priority/id", "Priority", groupTicket, "ref", "priorities", "", ""},
	{"priority/name", "Priority (by name)", groupTicket, "string", "", "", ""},
	{"board/id", "Board", groupTicket, "ref", "boards", "", ""},
	{"owner/id", "Owner", groupTicket, "ref", "members", "", ""},
	{"owner/identifier", "Owner (by identifier)", groupTicket, "identifier", "members", "", ""},
	{"resources", "Resources", groupTicket, "list", "members", "Comma-separated resource identifiers", ""},
	{"company/id", "Company", groupTicket, "ref", "companies", "", ""},
	{"company/identifier", "Company (by identifier)", groupTicket, "string", "", "", ""},
	{"contact/id", "Contact", groupTicket, "ref", "contacts", "", ""},
	{"contact/name", "Contact (by name)", groupTicket, "string", "", "", ""},
	{"type/name", "Type", groupTicket, "string", "", "", ""},
	{"subType/name", "Subtype", groupTicket, "string", "", "", ""},
	{"item/name", "Item", groupTicket, "string", "", "", ""},
	{"closedFlag", "Closed", groupTicket, "bool", "", "", ""},
	{"_info/updatedBy", "Last updated by", groupTicket, "identifier", "members", "Member who made this update", ""},

	// Latest note
	{"newNote", "A new note arrived", groupNote, "bool", "", "True when this update added a note, not just a field change", ""},
	{"latestNote/text", "Note text", groupNote, "string", "", "", ""},
	{"latestNote/internalAnalysisFlag", "Note is internal", groupNote, "bool", "", "", ""},
	{"latestNote/member/identifier", "Note author (member)", groupNote, "identifier", "members", "", ""},
	{"latestNote/contact/id", "Note author (contact)", groupNote, "ref", "contacts", "Use \"is not empty\" for any customer reply", ""},

	// What changed
	{"changed/status", "Status changed", groupChanged, "bool", "", "", ""},
	{"changed/priority", "Priority changed", groupChanged, "bool", "", "", ""},
	{"changed/owner", "Owner changed", groupChanged, "bool", "", "", ""},
	{"changed/resources", "Resources changed", groupChanged, "bool", "", "", ""},
	{"changed/board", "Moved boards", groupChanged, "bool", "", "", ""},
	{"changed/company", "Company changed", groupChanged, "bool", "", "", ""},
	{"changed/contact", "Contact changed", groupChanged, "bool", "", "", ""},
	{"changed/summary", "Summary changed", groupChanged, "bool", "", "", ""},
	{"changed/type", "Type changed", groupChanged, "bool", "", "", ""},
	{"changed/subType", "Subtype changed", groupChanged, "bool", "", "", ""},
	{"changed/item", "Item changed", groupChanged, "bool", "", "", ""},
	{"changed/closedFlag", "Opened or closed", groupChanged, "bool", "", "", ""},

	// Previous values (only set when the field changed in this update)
	{"old/status/id", "Previous status", groupPrevious, "ref", "statuses", "", ""},
	{"old/status/name", "Previous status (by name)", groupPrevious, "string", "", "", ""},
	{"old/priority/id", "Previous priority", groupPrevious, "ref", "priorities", "", ""},
	{"old/priority/name", "Previous priority (by name)", groupPrevious, "string", "", "", ""},
	{"old/owner/id", "Previous owner", groupPrevious, "ref", "members", "", ""},
	{"old/board/id", "Previous board", groupPrevious, "ref", "boards", "", ""},
	{"old/company/id", "Previous company", groupPrevious, "ref", "companies", "", ""},
	{"old/contact/id", "Previous contact", groupPrevious, "ref", "contacts", "", ""},
	{"old/resources", "Previous resources", groupPrevious, "list", "members", "", ""},
	{"old/summary", "Previous summary", groupPrevious, "string", "", "", ""},
	{"old/closedFlag", "Previously closed", groupPrevious, "bool", "", "", ""},
}
