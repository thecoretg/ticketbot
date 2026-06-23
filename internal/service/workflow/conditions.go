package workflow

import (
	"fmt"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// maxConditionDepth bounds the recursion of a workflow's condition tree to guard
// against pathological or malicious payloads.
const maxConditionDepth = 20

// EvalCtx is the data a condition is evaluated against: the live ticket plus its
// most recent note (nil when the ticket has none).
type EvalCtx struct {
	Ticket   *psa.Ticket
	LastNote *psa.ServiceTicketNote
}

// conditionFields is the allowlist of attributes a leaf condition can match
// against. Adding a new filterable field = add one entry here (and the matching
// option in the admin UI).
var conditionFields = map[string]func(EvalCtx) string{
	"summary":            func(c EvalCtx) string { return c.Ticket.Summary },
	"company_name":       func(c EvalCtx) string { return c.Ticket.Company.Name },
	"company_identifier": func(c EvalCtx) string { return c.Ticket.Company.Identifier },
	"contact_name":       func(c EvalCtx) string { return c.Ticket.Contact.Name },
	"status_name":        func(c EvalCtx) string { return c.Ticket.Status.Name },
	"board_name":         func(c EvalCtx) string { return c.Ticket.Board.Name },
	"type_name":          func(c EvalCtx) string { return c.Ticket.Type.Name },
	"subtype_name":       func(c EvalCtx) string { return c.Ticket.SubType.Name },
	"priority_name":      func(c EvalCtx) string { return c.Ticket.Priority.Name },
	"source_name":        func(c EvalCtx) string { return c.Ticket.Source.Name },
	"last_note_text": func(c EvalCtx) string {
		if c.LastNote != nil {
			return c.LastNote.Text
		}
		return ""
	},
	"last_note_sender": func(c EvalCtx) string {
		if c.LastNote == nil {
			return ""
		}
		if c.LastNote.Member.Name != "" {
			return c.LastNote.Member.Name
		}
		return c.LastNote.Contact.Name
	},
	// last_note_type yields the comma-joined set of types the note carries (a note
	// can be several at once), matched with the is_any_of / is_none_of operators.
	"last_note_type": func(c EvalCtx) string {
		if c.LastNote == nil {
			return ""
		}
		var types []string
		if c.LastNote.DetailDescriptionFlag {
			types = append(types, "discussion")
		}
		if c.LastNote.InternalFlag || c.LastNote.InternalAnalysisFlag {
			types = append(types, "internal")
		}
		if c.LastNote.ResolutionFlag {
			types = append(types, "resolution")
		}
		return strings.Join(types, ",")
	},
}

// conditionOperators is the allowlist of comparison operators. is_any_of /
// is_none_of treat both sides as comma-separated token sets (used for
// last_note_type, but valid on any field).
var conditionOperators = map[string]struct{}{
	"contains":     {},
	"not_contains": {},
	"equals":       {},
	"not_equals":   {},
	"starts_with":  {},
	"ends_with":    {},
	"is_any_of":    {},
	"is_none_of":   {},
}

// evalGroup reports whether a condition group matches. A nil group or one with no
// children matches everything (a workflow with no conditions always runs). The
// group operator combines children: "or" matches if any child matches, "and"
// (the default) matches only if all children match.
func evalGroup(g *models.ConditionGroup, c EvalCtx) bool {
	if g == nil || len(g.Children) == 0 {
		return true
	}

	if g.Operator == models.GroupOpOr {
		for _, n := range g.Children {
			if evalNode(n, c) {
				return true
			}
		}
		return false
	}

	for _, n := range g.Children {
		if !evalNode(n, c) {
			return false
		}
	}
	return true
}

// evalNode evaluates a single child: either a nested group or a leaf condition.
func evalNode(n models.ConditionNode, c EvalCtx) bool {
	switch {
	case n.Group != nil:
		return evalGroup(n.Group, c)
	case n.Condition != nil:
		return evalLeaf(*n.Condition, c)
	default:
		return false
	}
}

// evalLeaf reports whether a single leaf condition matches the ticket. String
// comparisons are case-insensitive. An unknown field or operator never matches
// (these are rejected at save time, so this is only a defensive guard).
func evalLeaf(cond models.Condition, c EvalCtx) bool {
	getter, ok := conditionFields[cond.Field]
	if !ok {
		return false
	}
	actual := strings.ToLower(getter(c))
	want := strings.ToLower(cond.Value)

	switch cond.Operator {
	case "contains":
		return strings.Contains(actual, want)
	case "not_contains":
		return !strings.Contains(actual, want)
	case "equals":
		return actual == want
	case "not_equals":
		return actual != want
	case "starts_with":
		return strings.HasPrefix(actual, want)
	case "ends_with":
		return strings.HasSuffix(actual, want)
	case "is_any_of":
		return tokensIntersect(actual, want)
	case "is_none_of":
		return !tokensIntersect(actual, want)
	default:
		return false
	}
}

// tokensIntersect reports whether the comma-separated token sets a and b share
// any value (case/space-insensitive). Used by the is_any_of / is_none_of ops so a
// single value (e.g. status "New") or a multi-valued one (note types
// "internal,discussion") both compare correctly against a selected set.
func tokensIntersect(a, b string) bool {
	set := make(map[string]struct{})
	for part := range strings.SplitSeq(a, ",") {
		if p := strings.TrimSpace(part); p != "" {
			set[p] = struct{}{}
		}
	}
	for part := range strings.SplitSeq(b, ",") {
		if _, ok := set[strings.TrimSpace(part)]; ok {
			return true
		}
	}
	return false
}

// validateGroup recursively checks that every leaf references a known field and
// operator, every group has a valid operator, every node sets exactly one of
// condition/group, and the tree does not exceed the depth limit.
func validateGroup(g *models.ConditionGroup, depth int) error {
	if g == nil {
		return nil
	}
	if depth > maxConditionDepth {
		return fmt.Errorf("condition tree exceeds max depth %d", maxConditionDepth)
	}
	switch g.Operator {
	case models.GroupOpAnd, models.GroupOpOr, "":
	default:
		return fmt.Errorf("invalid group operator %q", g.Operator)
	}

	for i, n := range g.Children {
		switch {
		case n.Group != nil && n.Condition != nil:
			return fmt.Errorf("child %d: cannot set both a condition and a group", i+1)
		case n.Group != nil:
			if err := validateGroup(n.Group, depth+1); err != nil {
				return err
			}
		case n.Condition != nil:
			if _, ok := conditionFields[n.Condition.Field]; !ok {
				return fmt.Errorf("child %d: unknown field %q", i+1, n.Condition.Field)
			}
			if _, ok := conditionOperators[n.Condition.Operator]; !ok {
				return fmt.Errorf("child %d: unknown operator %q", i+1, n.Condition.Operator)
			}
		default:
			return fmt.Errorf("child %d: must set either a condition or a group", i+1)
		}
	}
	return nil
}
