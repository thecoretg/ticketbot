package models

import (
	"errors"
	"fmt"
	"time"
)

// ListItemType names what a list holds. Types are registered in ListItemTypes; adding one there
// (plus a resolver in the lists service) is all a new kind of list needs.
type ListItemType string

const (
	ListItemContact ListItemType = "contact"
	ListItemCompany ListItemType = "company"
)

// ListItemTypeInfo describes one registered item type. Source is the lookup name shared with the
// condition builder (ConditionField.Source) and the /cw/{source} search endpoints.
type ListItemTypeInfo struct {
	Type   ListItemType `json:"type"`
	Label  string       `json:"label"`
	Plural string       `json:"plural"`
	Source string       `json:"source"`
	// DetailLabel names the secondary context shown next to each item (ListItem.Detail), such
	// as a contact's company. Empty when the type has none.
	DetailLabel string `json:"detail_label,omitempty"`
}

// ListItemTypes is served by GET /lists/types.
var ListItemTypes = []ListItemTypeInfo{
	{Type: ListItemContact, Label: "Contact", Plural: "Contacts", Source: "contacts", DetailLabel: "Company"},
	{Type: ListItemCompany, Label: "Company", Plural: "Companies", Source: "companies"},
}

func (t ListItemType) Valid() bool {
	_, ok := t.Info()
	return ok
}

func (t ListItemType) Info() (ListItemTypeInfo, bool) {
	for _, i := range ListItemTypes {
		if i.Type == t {
			return i, true
		}
	}
	return ListItemTypeInfo{}, false
}

// ListItemTypeForSource maps a condition field's Source ("contacts") to the list type it can be
// checked against ("contact").
func ListItemTypeForSource(source string) (ListItemType, bool) {
	for _, i := range ListItemTypes {
		if i.Source == source {
			return i.Type, true
		}
	}
	return "", false
}

var (
	ErrListNotFound     = errors.New("list not found")
	ErrListNameTaken    = errors.New("a list with that name already exists")
	ErrListItemNotFound = errors.New("item is not in this list")
)

// List is an admin-defined set of items that conditions reference as `path in list <id>`.
type List struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	ItemType    ListItemType `json:"item_type"`
	Description string       `json:"description"`
	ItemCount   int          `json:"item_count"`
	CreatedOn   time.Time    `json:"created_on"`
	UpdatedOn   time.Time    `json:"updated_on"`
}

// ListItem is one member. Label is resolved from the synced ConnectWise tables; Missing is set
// when the item is no longer there (deleted or never synced).
type ListItem struct {
	ListID  int       `json:"list_id"`
	ItemID  int       `json:"item_id"`
	Label   string    `json:"label"`
	Detail  string    `json:"detail,omitempty"` // see ListItemTypeInfo.DetailLabel
	Missing bool      `json:"missing,omitempty"`
	AddedOn time.Time `json:"added_on"`
}

// ListMembership is a (list, item) pair, the shape the workflow engine loads.
type ListMembership struct {
	ListID int
	ItemID int
}

// ListReference points at a workflow node whose condition uses a list.
type ListReference struct {
	WorkflowID   int    `json:"workflow_id"`
	WorkflowName string `json:"workflow_name"`
	BoardName    string `json:"board_name"`
	NodeID       string `json:"node_id"`
	NodeTitle    string `json:"node_title"`
}

// ListDetail is a list with its members and the rules that reference it.
type ListDetail struct {
	List
	Items  []ListItem      `json:"items"`
	UsedBy []ListReference `json:"used_by"`
}

// ListValidationError is a bad create/update payload.
type ListValidationError struct {
	Field   string
	Message string
}

func (e *ListValidationError) Error() string {
	return e.Field + ": " + e.Message
}

// ListInUseError blocks deleting a list that workflow rules still reference.
type ListInUseError struct {
	Refs []ListReference
}

func (e *ListInUseError) Error() string {
	return fmt.Sprintf("list is used by %d workflow rule(s)", len(e.Refs))
}
