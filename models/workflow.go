package models

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrWorkflowNotFound       = errors.New("workflow not found")
	ErrWorkflowExistsForBoard = errors.New("a workflow already exists for this board")
)

// Trigger decides whether a rule runs for a newly seen ticket, an updated one, or both.
type Trigger string

const (
	TriggerCreate Trigger = "create"
	TriggerUpdate Trigger = "update"
	TriggerBoth   Trigger = "both"
)

func (t Trigger) Valid() bool {
	switch t {
	case TriggerCreate, TriggerUpdate, TriggerBoth:
		return true
	}
	return false
}

// Matches reports whether the trigger fires for a ticket that is (or is not) new.
func (t Trigger) Matches(isNew bool) bool {
	switch t {
	case TriggerCreate:
		return isNew
	case TriggerUpdate:
		return !isNew
	case TriggerBoth:
		return true
	}
	return false
}

type ActionKind string

const (
	ActionNotify     ActionKind = "notify"
	ActionAddNote    ActionKind = "add_note"
	ActionSkipNotify ActionKind = "skip_notify"
)

type NotifyTarget string

const (
	TargetRoom           NotifyTarget = "room"
	TargetPerson         NotifyTarget = "person"
	TargetResourcesOwner NotifyTarget = "resources_owner"
)

// Workflow is the per-board rule chain. Rules are stored as JSON and replaced as a whole.
type Workflow struct {
	ID        int       `json:"id"`
	BoardID   int       `json:"board_id"`
	BoardName string    `json:"board_name,omitempty"`
	Name      string    `json:"name"`
	Enabled   bool      `json:"enabled"`
	DryRun    bool      `json:"dry_run"`
	Rules     []Rule    `json:"rules"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

// Rule is one step in a workflow, evaluated in order.
type Rule struct {
	// ID is a server-assigned UUID so events can reference a rule across renames and reorders.
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Enabled        bool     `json:"enabled"`
	Trigger        Trigger  `json:"trigger"`
	Condition      string   `json:"condition"` // cwquery source; empty always matches
	Actions        []Action `json:"actions"`
	StopProcessing bool     `json:"stop_processing"`
}

// Action is a tagged union: exactly the sub-struct matching Kind is set.
type Action struct {
	Kind    ActionKind     `json:"kind"`
	Enabled bool           `json:"enabled"`
	Notify  *NotifyAction  `json:"notify,omitempty"`
	AddNote *AddNoteAction `json:"add_note,omitempty"`
}

type NotifyAction struct {
	Target      NotifyTarget `json:"target"`
	RecipientID *int         `json:"recipient_id,omitempty"` // webex_recipient.id for room / person
}

// AddNoteAction posts a plain-text note to the ticket. At least one flag must be set.
type AddNoteAction struct {
	Text       string `json:"text"`
	Internal   bool   `json:"internal"`   // InternalAnalysisFlag
	Discussion bool   `json:"discussion"` // DetailDescriptionFlag
	Resolution bool   `json:"resolution"` // ResolutionFlag
}

// ValidationError points at one problem in a workflow document.
type ValidationError struct {
	RuleIndex   int    `json:"rule_index"`
	RuleID      string `json:"rule_id,omitempty"`
	ActionIndex *int   `json:"action_index,omitempty"`
	Field       string `json:"field"`
	Message     string `json:"message"`
	Pos         *int   `json:"pos,omitempty"` // character offset for condition syntax errors
}

func (e ValidationError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "rule %d", e.RuleIndex+1)
	if e.ActionIndex != nil {
		fmt.Fprintf(&sb, " action %d", *e.ActionIndex+1)
	}
	fmt.Fprintf(&sb, ": %s: %s", e.Field, e.Message)
	return sb.String()
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	msgs := make([]string, 0, len(v))
	for _, e := range v {
		msgs = append(msgs, e.Error())
	}
	return "workflow validation failed: " + strings.Join(msgs, "; ")
}
