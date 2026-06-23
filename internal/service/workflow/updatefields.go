package workflow

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// patchKind describes how a field's single templated string value is turned into
// the JSON value Connectwise expects in a PATCH operation.
type patchKind string

const (
	// kindText: the value is a plain string (e.g. summary).
	kindText patchKind = "text"
	// kindRefID: the value is a numeric id wrapped as {"id": <n>} (e.g. board, status).
	kindRefID patchKind = "ref_id"
	// kindRefIdentifier: the value is wrapped as {"identifier": "<v>"} (e.g. owner, company by identifier).
	kindRefIdentifier patchKind = "ref_identifier"
	// kindRefName: the value is wrapped as {"name": "<v>"} (e.g. priority, type — not synced locally).
	kindRefName patchKind = "ref_name"
)

// cwSummaryMaxLen is Connectwise's hard limit on ticket summary length.
const cwSummaryMaxLen = 100

// updateField is one entry in the allowlist of ticket fields a ticket_update
// action may mutate. The catalog is the single source of truth for the builder
// UI, the save-time validation, and run-time idempotency. Adding a new updatable
// field = add one entry to updateFieldList (and, if it has a local picker source,
// expose the matching list endpoint the frontend reads).
type updateField struct {
	Path        string                   // canonical CW patch path for replace, e.g. "summary", "company"
	Label       string                   // human label for the UI
	Kind        patchKind                // how the value string maps to a CW value
	Picker      string                   // picker source: "company"|"contact"|"member"|"board"|"status"|"" (free text)
	AllowRemove bool                     // whether a remove op is offered
	RemovePath  string                   // path used for a remove op; defaults to Path
	MaxLen      int                      // 0 = no limit; otherwise truncate (e.g. summary)
	DependsOn   string                   // a field whose value scopes this one's picker (contact→company, status→board)
	Requires    string                   // a field that must also be set when this one is (board→status)
	Current     func(*psa.Ticket) string // current comparable value, for idempotency
}

// removePath returns the path to use for a remove op.
func (f updateField) removePath() string {
	if f.RemovePath != "" {
		return f.RemovePath
	}
	return f.Path
}

// buildValue turns a rendered string into the JSON value Connectwise expects for
// this field's kind.
func (f updateField) buildValue(rendered string) (any, error) {
	switch f.Kind {
	case kindText:
		return rendered, nil
	case kindRefIdentifier:
		return map[string]any{"identifier": rendered}, nil
	case kindRefName:
		return map[string]any{"name": rendered}, nil
	case kindRefID:
		n, err := strconv.Atoi(strings.TrimSpace(rendered))
		if err != nil {
			return nil, fmt.Errorf("value %q is not a numeric id", rendered)
		}
		return map[string]any{"id": n}, nil
	default:
		return nil, fmt.Errorf("unknown patch kind %q", f.Kind)
	}
}

// updateFieldList is the allowlist of updatable ticket fields. Fields with a
// local picker source store the value the picker resolves to (company/board/
// status → id, owner → member identifier); fields without one (priority/type/
// subType are not synced locally) accept a templated free-text name.
var updateFieldList = []updateField{
	{Path: "summary", Label: "Summary", Kind: kindText, MaxLen: cwSummaryMaxLen,
		Current: func(t *psa.Ticket) string { return t.Summary }},
	{Path: "company", Label: "Company", Kind: kindRefIdentifier, Picker: "company",
		Current: func(t *psa.Ticket) string { return t.Company.Identifier }},
	{Path: "contact", Label: "Contact", Kind: kindRefID, Picker: "contact", DependsOn: "company",
		Current: func(t *psa.Ticket) string { return itoa(t.Contact.ID) }},
	{Path: "owner", Label: "Owner", Kind: kindRefIdentifier, Picker: "member", AllowRemove: true, RemovePath: "owner/id",
		Current: func(t *psa.Ticket) string { return t.Owner.Identifier }},
	{Path: "board", Label: "Board", Kind: kindRefID, Picker: "board", Requires: "status",
		Current: func(t *psa.Ticket) string { return itoa(t.Board.ID) }},
	{Path: "status", Label: "Status", Kind: kindRefID, Picker: "status", DependsOn: "board",
		Current: func(t *psa.Ticket) string { return itoa(t.Status.ID) }},
	{Path: "priority", Label: "Priority", Kind: kindRefName,
		Current: func(t *psa.Ticket) string { return t.Priority.Name }},
	{Path: "type", Label: "Type", Kind: kindRefName,
		Current: func(t *psa.Ticket) string { return t.Type.Name }},
	{Path: "subType", Label: "Subtype", Kind: kindRefName,
		Current: func(t *psa.Ticket) string { return t.SubType.Name }},
}

// updateFieldByPath indexes the catalog by path for lookup.
var updateFieldByPath = func() map[string]updateField {
	m := make(map[string]updateField, len(updateFieldList))
	for _, f := range updateFieldList {
		m[f.Path] = f
	}
	return m
}()

// UpdateFieldInfo is the UI-facing description of an updatable field, returned by
// the update-fields endpoint so the admin panel can render the op builder.
type UpdateFieldInfo struct {
	Path        string `json:"path"`
	Label       string `json:"label"`
	Kind        string `json:"kind"`         // text | ref_id | ref_identifier | ref_name
	Picker      string `json:"picker"`       // company | contact | member | board | status | "" (free text)
	AllowRemove bool   `json:"allow_remove"` // whether a remove op is offered
	DependsOn   string `json:"depends_on"`   // field that scopes this picker (contact→company, status→board)
	Requires    string `json:"requires"`     // field that must also be set (board→status)
}

// UpdateFields returns the catalog of updatable fields for the admin UI.
func UpdateFields() []UpdateFieldInfo {
	out := make([]UpdateFieldInfo, 0, len(updateFieldList))
	for _, f := range updateFieldList {
		out = append(out, UpdateFieldInfo{
			Path:        f.Path,
			Label:       f.Label,
			Kind:        string(f.Kind),
			Picker:      f.Picker,
			AllowRemove: f.AllowRemove,
			DependsOn:   f.DependsOn,
			Requires:    f.Requires,
		})
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return ""
	}
	return strconv.Itoa(n)
}
