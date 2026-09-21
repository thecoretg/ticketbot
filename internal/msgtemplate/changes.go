package msgtemplate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

// changeLabels are the human names of the diffed fields, for the "what changed" line.
var changeLabels = map[string]string{
	ticketdiff.FieldSummary:    "Summary",
	ticketdiff.FieldStatus:     "Status",
	ticketdiff.FieldBoard:      "Board",
	ticketdiff.FieldOwner:      "Owner",
	ticketdiff.FieldResources:  "Resources",
	ticketdiff.FieldPriority:   "Priority",
	ticketdiff.FieldType:       "Type",
	ticketdiff.FieldSubType:    "Subtype",
	ticketdiff.FieldItem:       "Item",
	ticketdiff.FieldCompany:    "Company",
	ticketdiff.FieldContact:    "Contact",
	ticketdiff.FieldClosedFlag: "Closed",
}

// FormatChanges renders a change list as "Status: New → Assigned · Owner: — → jdoe". It is the
// {{changes}} placeholder and the default message's Changed line; empty when nothing changed.
func FormatChanges(changes []models.FieldChange) string {
	parts := make([]string, 0, len(changes))
	for _, c := range changes {
		label := changeLabels[c.Field]
		if label == "" {
			label = c.Field
		}
		parts = append(parts, fmt.Sprintf("%s: %s → %s", label, changeValue(c.Old), changeValue(c.New)))
	}
	return strings.Join(parts, " · ")
}

// changeValue prints one side of a change: a reference by name, a resource list joined, a flag as
// yes / no, nothing as a dash. Values that went through JSON (history rows) arrive as maps.
func changeValue(v any) string {
	switch x := v.(type) {
	case nil:
		return "—"
	case ticketdiff.Ref:
		if x.Name != "" {
			return x.Name
		}
		if x.ID == 0 {
			return "—"
		}
		return fmt.Sprintf("#%d", x.ID)
	case *ticketdiff.Ref:
		if x == nil {
			return "—"
		}
		return changeValue(*x)
	case map[string]any:
		if n, ok := x["name"].(string); ok && n != "" {
			return n
		}
		if id, ok := x["id"].(float64); ok && id != 0 {
			return fmt.Sprintf("#%d", int(id))
		}
		return "—"
	case []string:
		if len(x) == 0 {
			return "—"
		}
		s := append([]string(nil), x...)
		sort.Strings(s)
		return strings.Join(s, ", ")
	case []any:
		if len(x) == 0 {
			return "—"
		}
		s := make([]string, 0, len(x))
		for _, e := range x {
			s = append(s, fmt.Sprint(e))
		}
		sort.Strings(s)
		return strings.Join(s, ", ")
	case bool:
		if x {
			return "yes"
		}
		return "no"
	case string:
		if strings.TrimSpace(x) == "" {
			return "—"
		}
		return x
	}
	return fmt.Sprint(v)
}
