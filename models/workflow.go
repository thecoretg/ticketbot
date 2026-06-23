package models

import (
	"encoding/json"
	"errors"
	"time"
)

var ErrWorkflowNotFound = errors.New("workflow not found")

// Workflow action types. These are the registry keys persisted inside each
// element of workflow.actions and dispatched on at run time.
const (
	WorkflowActionTicketUpdate      = "ticket_update"      // patch ticket fields via the CW API
	WorkflowActionAddNote           = "add_note"           // post a note to the ticket
	WorkflowActionSendMessage       = "send_message"       // send a Webex message to a recipient
	WorkflowActionSkipNotifications = "skip_notifications" // suppress the default notifier for this ticket
	WorkflowActionAddResource       = "add_resource"       // add a member as a ticket resource
	WorkflowActionAddEmailCc        = "add_email_cc"        // add an email address to the ticket's CC list
)

// On-ticket-action values control which webhook events a workflow fires on.
const (
	WorkflowOnNew     = "new"     // only the first time the bot sees a ticket
	WorkflowOnUpdated = "updated" // only on subsequent updates
	WorkflowOnBoth    = "both"    // always
)

// Boolean operators for a ConditionGroup.
const (
	GroupOpAnd = "and"
	GroupOpOr  = "or"
)

// Condition is a single leaf match predicate evaluated against a ticket. Field
// and Operator are validated against the workflow service's allowlists at save
// time.
type Condition struct {
	Field    string `json:"field"`    // e.g. "summary", "last_note_text"
	Operator string `json:"operator"` // e.g. "contains", "equals"
	Value    string `json:"value"`
}

// ConditionNode is one child of a ConditionGroup. Exactly one of Condition or
// Group is set; this is enforced at save time.
type ConditionNode struct {
	Condition *Condition      `json:"condition,omitempty"`
	Group     *ConditionGroup `json:"group,omitempty"`
}

// ConditionGroup is a nested boolean node: an operator applied over an ordered
// list of children. A nil group or one with no children matches everything.
type ConditionGroup struct {
	Operator string          `json:"operator"` // "and" | "or"
	Children []ConditionNode `json:"children"`
}

// Action is one step in a workflow. Config holds the action-specific parameters
// (with Go-template string values where supported) as a raw JSON object; the
// service unmarshals it per-action type.
type Action struct {
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

// Workflow is one configurable automation that runs against a Connectwise ticket
// before it is synced locally. A workflow fires when its board, on-ticket-action
// trigger, and condition tree all match, then runs its actions in order.
type Workflow struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	CwBoardID      int             `json:"cw_board_id"`          // required: workflow is scoped to one board
	OnTicketAction string          `json:"on_ticket_action"`     // new | updated | both
	Root           *ConditionGroup `json:"conditions,omitempty"` // optional root group; nil/empty = always match
	Actions        []Action        `json:"actions"`              // ordered, at least one
	Priority       int             `json:"priority"`             // lower runs first
	Enabled        bool            `json:"enabled"`
	CreatedOn      time.Time       `json:"created_on"`
	UpdatedOn      time.Time       `json:"updated_on"`
}
