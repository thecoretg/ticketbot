package transformer

import (
	"fmt"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// conditionFields is the allowlist of ticket attributes a rule condition can
// match against. Adding a new filterable field = add one entry here (and the
// matching option in the admin UI).
var conditionFields = map[string]func(*psa.Ticket) string{
	"summary":            func(t *psa.Ticket) string { return t.Summary },
	"company_name":       func(t *psa.Ticket) string { return t.Company.Name },
	"company_identifier": func(t *psa.Ticket) string { return t.Company.Identifier },
	"contact_name":       func(t *psa.Ticket) string { return t.Contact.Name },
	"status_name":        func(t *psa.Ticket) string { return t.Status.Name },
	"board_name":         func(t *psa.Ticket) string { return t.Board.Name },
	"type_name":          func(t *psa.Ticket) string { return t.Type.Name },
	"subtype_name":       func(t *psa.Ticket) string { return t.SubType.Name },
	"priority_name":      func(t *psa.Ticket) string { return t.Priority.Name },
	"source_name":        func(t *psa.Ticket) string { return t.Source.Name },
}

// conditionOperators is the allowlist of comparison operators.
var conditionOperators = map[string]struct{}{
	"contains":     {},
	"not_contains": {},
	"equals":       {},
	"not_equals":   {},
	"starts_with":  {},
	"ends_with":    {},
}

// evalCondition reports whether a single condition matches the ticket. String
// comparisons are case-insensitive. An unknown field or operator never matches
// (these are rejected at save time, so this is only a defensive guard).
func evalCondition(c models.RuleCondition, t *psa.Ticket) bool {
	getter, ok := conditionFields[c.Field]
	if !ok {
		return false
	}
	actual := strings.ToLower(getter(t))
	want := strings.ToLower(c.Value)

	switch c.Operator {
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
	default:
		return false
	}
}

// validateConditions checks every condition references a known field and operator.
func validateConditions(conds []models.RuleCondition) error {
	for i, c := range conds {
		if _, ok := conditionFields[c.Field]; !ok {
			return fmt.Errorf("condition %d: unknown field %q", i+1, c.Field)
		}
		if _, ok := conditionOperators[c.Operator]; !ok {
			return fmt.Errorf("condition %d: unknown operator %q", i+1, c.Operator)
		}
	}
	return nil
}
