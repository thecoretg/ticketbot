// Package ticketdiff compares two ConnectWise tickets over a curated set of fields and reports
// what changed. It is the basis for "only log / act when something actually changed".
package ticketdiff

import (
	"slices"
	"sort"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Field names used in FieldChange.Field and in cwquery's `changed/<field>` pseudo-field.
const (
	FieldSummary    = "summary"
	FieldStatus     = "status"
	FieldBoard      = "board"
	FieldOwner      = "owner"
	FieldResources  = "resources"
	FieldPriority   = "priority"
	FieldType       = "type"
	FieldSubType    = "subType"
	FieldItem       = "item"
	FieldCompany    = "company"
	FieldContact    = "contact"
	FieldClosedFlag = "closedFlag"
)

// CuratedFields is every field Diff knows how to compare.
var CuratedFields = []string{
	FieldSummary, FieldStatus, FieldBoard, FieldOwner, FieldResources, FieldPriority,
	FieldType, FieldSubType, FieldItem, FieldCompany, FieldContact, FieldClosedFlag,
}

// LegacyFields are the fields that can be reconstructed from a pre-v2 cw_ticket row (no raw JSON).
var LegacyFields = []string{
	FieldSummary, FieldStatus, FieldBoard, FieldOwner, FieldResources, FieldCompany, FieldContact,
}

// Ref is the {id, name} shape used for reference fields in FieldChange.Old / New.
type Ref struct {
	ID   int    `json:"id"`
	Name string `json:"name,omitempty"`
}

// Diff returns the fields (from fields) whose values differ between old and cur.
// A nil old ticket produces no changes; callers should treat "no stored ticket" as created.
func Diff(old, cur *psa.Ticket, fields []string) []models.FieldChange {
	if old == nil || cur == nil {
		return nil
	}

	var out []models.FieldChange
	for _, f := range fields {
		o, n, ok := values(old, cur, f)
		if !ok {
			continue
		}
		if !equalValues(o, n) {
			out = append(out, models.FieldChange{Field: f, Old: o, New: n})
		}
	}

	return out
}

// FieldNames extracts the field names from a change list, for cwquery.NewDocument.
func FieldNames(changes []models.FieldChange) []string {
	names := make([]string, 0, len(changes))
	for _, c := range changes {
		names = append(names, c.Field)
	}
	return names
}

// FromStored reconstructs the comparable parts of a psa.Ticket from a stored row. It is used for
// rows written before raw JSON was stored; only LegacyFields are meaningful on the result.
func FromStored(t *models.Ticket) *psa.Ticket {
	if t == nil {
		return nil
	}

	p := &psa.Ticket{ID: t.ID, Summary: t.Summary, ClosedFlag: t.ClosedFlag}
	p.Board.ID = t.BoardID
	p.Status.ID = t.StatusID
	p.Company.ID = t.CompanyID
	if t.OwnerID != nil {
		p.Owner.ID = *t.OwnerID
	}
	if t.ContactID != nil {
		p.Contact.ID = *t.ContactID
	}
	if t.Resources != nil {
		p.Resources = *t.Resources
	}
	if t.PriorityID != nil {
		p.Priority.ID = *t.PriorityID
	}
	if t.PriorityName != nil {
		p.Priority.Name = *t.PriorityName
	}
	if t.TypeID != nil {
		p.Type.ID = *t.TypeID
	}
	if t.TypeName != nil {
		p.Type.Name = *t.TypeName
	}
	if t.SubTypeID != nil {
		p.SubType.ID = *t.SubTypeID
	}
	if t.SubTypeName != nil {
		p.SubType.Name = *t.SubTypeName
	}
	if t.ItemID != nil {
		p.Item.ID = *t.ItemID
	}
	if t.ItemName != nil {
		p.Item.Name = *t.ItemName
	}

	return p
}

func values(old, cur *psa.Ticket, field string) (o, n any, ok bool) {
	switch field {
	case FieldSummary:
		return old.Summary, cur.Summary, true
	case FieldStatus:
		return Ref{old.Status.ID, old.Status.Name}, Ref{cur.Status.ID, cur.Status.Name}, true
	case FieldBoard:
		return Ref{old.Board.ID, old.Board.Name}, Ref{cur.Board.ID, cur.Board.Name}, true
	case FieldOwner:
		return Ref{old.Owner.ID, old.Owner.Name}, Ref{cur.Owner.ID, cur.Owner.Name}, true
	case FieldResources:
		return Resources(old.Resources), Resources(cur.Resources), true
	case FieldPriority:
		return Ref{old.Priority.ID, old.Priority.Name}, Ref{cur.Priority.ID, cur.Priority.Name}, true
	case FieldType:
		return Ref{old.Type.ID, old.Type.Name}, Ref{cur.Type.ID, cur.Type.Name}, true
	case FieldSubType:
		return Ref{old.SubType.ID, old.SubType.Name}, Ref{cur.SubType.ID, cur.SubType.Name}, true
	case FieldItem:
		return Ref{old.Item.ID, old.Item.Name}, Ref{cur.Item.ID, cur.Item.Name}, true
	case FieldCompany:
		return Ref{old.Company.ID, old.Company.Name}, Ref{cur.Company.ID, cur.Company.Name}, true
	case FieldContact:
		return Ref{old.Contact.ID, old.Contact.Name}, Ref{cur.Contact.ID, cur.Contact.Name}, true
	case FieldClosedFlag:
		return old.ClosedFlag, cur.ClosedFlag, true
	}

	return nil, nil, false
}

// Resources splits ConnectWise's comma-separated resource string into a sorted, trimmed slice.
func Resources(s string) []string {
	if strings.TrimSpace(s) == "" {
		return []string{}
	}

	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)

	return out
}

// equalValues compares two values produced by values(). Refs compare by ID only, since a rename
// in ConnectWise of the same entity is not a ticket change.
func equalValues(a, b any) bool {
	switch av := a.(type) {
	case Ref:
		bv, ok := b.(Ref)
		return ok && av.ID == bv.ID
	case []string:
		bv, ok := b.([]string)
		return ok && slices.Equal(av, bv)
	default:
		return a == b
	}
}
