package models

import (
	"encoding/json"
	"time"
)

// EventKind classifies a ticket_event row.
type EventKind string

const (
	EventCreated      EventKind = "created"      // ticket first seen
	EventUpdated      EventKind = "updated"      // stored ticket differed from ConnectWise
	EventDeleted      EventKind = "deleted"      // ticket removed from ConnectWise
	EventLoopGuard    EventKind = "loop_guard"   // update authored by ticketbot itself; rules skipped
	EventWorkflow     EventKind = "workflow"     // one per run: which workflow ran and how each rule fared
	EventAction       EventKind = "action"       // one per action outcome
	EventNotification EventKind = "notification" // one per recipient
	EventError        EventKind = "error"        // intake aborted
)

// EventSource records what triggered an intake run.
type EventSource string

const (
	SourceWebhook EventSource = "webhook"
	SourceSync    EventSource = "sync"
	SourceManual  EventSource = "manual"
)

// TicketEvent is one entry in a ticket's activity thread.
type TicketEvent struct {
	ID         int64           `json:"id"`
	TicketID   int             `json:"ticket_id"`
	RunID      string          `json:"run_id"`
	Kind       EventKind       `json:"kind"`
	Source     EventSource     `json:"source"`
	DryRun     bool            `json:"dry_run"`
	Payload    json.RawMessage `json:"payload"`
	OccurredAt time.Time       `json:"occurred_at"`
}

// FieldChange is one differing field between the stored ticket and the fetched ticket.
// Old and New are scalars, {"id","name"} objects for references, or sorted []string for resources.
type FieldChange struct {
	Field string `json:"field"`
	Old   any    `json:"old"`
	New   any    `json:"new"`
}

// ChangePayload is the payload for created and updated events.
type ChangePayload struct {
	Changes   []FieldChange `json:"changes"`
	NewNote   *NotePayload  `json:"new_note,omitempty"`
	UpdatedBy string        `json:"updated_by,omitempty"`
}

type NotePayload struct {
	ID               int    `json:"id"`
	AuthorIdentifier string `json:"author_identifier,omitempty"`
	AuthorName       string `json:"author_name,omitempty"`
	Internal         bool   `json:"internal"`
	Preview          string `json:"preview"`
}

// LoopGuardPayload explains why rules were skipped for a self-authored update.
type LoopGuardPayload struct {
	Reason     string `json:"reason"` // note_author | updated_by (webhook_member in events recorded before 2026-09-18)
	Identifier string `json:"identifier"`
}

// WorkflowPayload summarizes a workflow run.
type WorkflowPayload struct {
	WorkflowID   *int                 `json:"workflow_id,omitempty"`
	WorkflowName string               `json:"workflow_name,omitempty"`
	Found        bool                 `json:"found"`
	Enabled      bool                 `json:"enabled"`
	Rules        []RuleOutcomePayload `json:"rules,omitempty"`
}

type RuleOutcomePayload struct {
	RuleID   string `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Matched  bool   `json:"matched"`
	Skipped  string `json:"skipped,omitempty"` // disabled | trigger
	Error    string `json:"error,omitempty"`
	Stopped  bool   `json:"stopped"`
}

// ActionPayload records one action's outcome.
type ActionPayload struct {
	RuleID   string         `json:"rule_id"`
	RuleName string         `json:"rule_name"`
	Index    int            `json:"index"`
	Kind     string         `json:"kind"`
	Result   string         `json:"result"` // ok | would_run | queued | skipped | error
	Reason   string         `json:"reason,omitempty"`
	Error    string         `json:"error,omitempty"`
	Output   map[string]any `json:"output,omitempty"`
}

// NotificationPayload records one recipient's outcome.
type NotificationPayload struct {
	RuleID         string   `json:"rule_id,omitempty"`
	RuleName       string   `json:"rule_name,omitempty"`
	RecipientID    int      `json:"recipient_id"`
	RecipientName  string   `json:"recipient_name"`
	RecipientType  string   `json:"recipient_type"`
	ForwardedFrom  []string `json:"forwarded_from,omitempty"`
	Result         string   `json:"result"` // sent | would_send | error
	Error          string   `json:"error,omitempty"`
	NotificationID *int     `json:"notification_id,omitempty"`
}

// ErrorPayload is the payload for error events.
type ErrorPayload struct {
	Stage string `json:"stage"`
	Error string `json:"error"`
}

// NewTicketEvent marshals payload and builds an event. A nil payload becomes {}.
func NewTicketEvent(ticketID int, runID string, kind EventKind, source EventSource, dryRun bool, payload any, at time.Time) (*TicketEvent, error) {
	raw := json.RawMessage("{}")
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw = b
	}

	return &TicketEvent{
		TicketID:   ticketID,
		RunID:      runID,
		Kind:       kind,
		Source:     source,
		DryRun:     dryRun,
		Payload:    raw,
		OccurredAt: at,
	}, nil
}
