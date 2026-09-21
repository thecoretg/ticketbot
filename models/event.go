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

// WorkflowPayload summarizes a workflow run: the steps every fired trigger's walk visited, in order.
type WorkflowPayload struct {
	WorkflowID   *int          `json:"workflow_id,omitempty"`
	WorkflowName string        `json:"workflow_name,omitempty"`
	Found        bool          `json:"found"`
	Enabled      bool          `json:"enabled"`
	Event        TriggerEvent  `json:"event,omitempty"`
	Steps        []StepPayload `json:"steps,omitempty"`
	// Conflicts lists fields two nodes set in this run; the later node's value was written.
	Conflicts []ConflictPayload `json:"conflicts,omitempty"`
}

// ConflictPayload names the two nodes that set the same ticket field.
type ConflictPayload struct {
	Path            string `json:"path"`
	SupersededID    string `json:"superseded_id"`
	SupersededTitle string `json:"superseded_title"`
	ByID            string `json:"by_id"`
	ByTitle         string `json:"by_title"`
}

// StepPayload records one node the run visited. Via is the edge the walk arrived by and Port the
// output it left by, so a canvas can light the path.
type StepPayload struct {
	NodeID  string   `json:"node_id"`
	Title   string   `json:"title"`
	No      int      `json:"no"`
	Kind    NodeKind `json:"kind"`
	Trigger string   `json:"trigger,omitempty"`
	Via     string   `json:"via,omitempty"`
	Port    Port     `json:"port,omitempty"`
	Matched *bool    `json:"matched,omitempty"`
	Skipped string   `json:"skipped,omitempty"` // disabled | joined
	Error   string   `json:"error,omitempty"`
}

// ActionPayload records one action node's outcome.
type ActionPayload struct {
	NodeID string         `json:"node_id"`
	Title  string         `json:"title"`
	No     int            `json:"no"`
	Kind   string         `json:"kind"`
	Result string         `json:"result"` // ok | would_run | queued | skipped | superseded | error
	Reason string         `json:"reason,omitempty"`
	Error  string         `json:"error,omitempty"`
	Output map[string]any `json:"output,omitempty"`
}

// NotificationPayload records one recipient's outcome.
type NotificationPayload struct {
	NodeID         string   `json:"node_id,omitempty"`
	Title          string   `json:"title,omitempty"`
	RecipientID    int      `json:"recipient_id"`
	RecipientName  string   `json:"recipient_name"`
	RecipientType  string   `json:"recipient_type"`
	ForwardedFrom  []string `json:"forwarded_from,omitempty"`
	RedirectedTo   string   `json:"redirected_to,omitempty"`
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
