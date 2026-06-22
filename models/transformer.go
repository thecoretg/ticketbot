package models

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrTransformerRuleNotFound = errors.New("transformer rule not found")

// Transformer action types. These are the registry keys persisted in
// transformer_rule.action and dispatched on at run time.
const (
	TransformerActionUpdateSummary = "update_summary"
	TransformerActionAddNote       = "add_note"
)

// Transformer apply-on values control which webhook events a rule fires on.
const (
	TransformerApplyNew     = "new"     // only the first time the bot sees a ticket
	TransformerApplyUpdated = "updated" // only on subsequent updates
	TransformerApplyBoth    = "both"    // always
)

// RuleCondition is a single match predicate evaluated against a ticket. A rule
// only runs when all of its conditions match (AND). Field and Operator are
// validated against the transformer service's allowlists at save time.
type RuleCondition struct {
	Field    string `json:"field"`    // e.g. "summary", "company_name"
	Operator string `json:"operator"` // e.g. "contains", "equals"
	Value    string `json:"value"`
}

// TransformerRule is one configurable step in the ticket transformer pipeline.
// Config holds action-specific parameters (with Go-template string values) as a
// raw JSON object; the service unmarshals it per-action.
type TransformerRule struct {
	ID         int             `json:"id"`
	Name       string          `json:"name"`
	Action     string          `json:"action"`
	CwBoardID  *int            `json:"cw_board_id"` // nil = applies to all boards
	Config     json.RawMessage `json:"config"`
	Conditions []RuleCondition `json:"conditions"` // all must match (AND)
	ApplyOn    string          `json:"apply_on"`
	Priority   int             `json:"priority"` // lower runs first
	Enabled    bool            `json:"enabled"`
	CreatedOn  time.Time       `json:"created_on"`
	UpdatedOn  time.Time       `json:"updated_on"`
}
