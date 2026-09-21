package models

import (
	"errors"
	"time"
)

var ErrRunNotFound = errors.New("workflow run not found")

// RunOutcome is the one-word verdict on a workflow run, for the results list.
type RunOutcome string

const (
	OutcomeClean          RunOutcome = "clean"           // every step ran, nothing failed
	OutcomeErrors         RunOutcome = "errors"          // an action, notification or the run itself failed
	OutcomeNobodyNotified RunOutcome = "nobody_notified" // notify steps ran but resolved to nobody
	OutcomeNoTrigger      RunOutcome = "no_trigger"      // the workflow was enabled but no trigger listened for the event
)

// WorkflowRun summarises one run for the results page. The ticket history events sharing its
// RunID are the detail.
type WorkflowRun struct {
	RunID          string       `json:"run_id"`
	TicketID       int          `json:"ticket_id"`
	BoardID        int          `json:"board_id"`
	BoardName      string       `json:"board_name,omitempty"`
	WorkflowID     *int         `json:"workflow_id,omitempty"`
	WorkflowName   string       `json:"workflow_name"`
	Event          TriggerEvent `json:"event"`
	Source         EventSource  `json:"source"`
	DryRun         bool         `json:"dry_run"`
	StartedAt      time.Time    `json:"started_at"`
	DurationMs     int          `json:"duration_ms"`
	Steps          int          `json:"steps"`
	Actions        int          `json:"actions"`
	Writes         int          `json:"writes"`
	NotifSent      int          `json:"notif_sent"`
	NotifWouldSend int          `json:"notif_would_send"`
	NotifNone      int          `json:"notif_none"`
	Errors         int          `json:"errors"`
	Outcome        RunOutcome   `json:"outcome"`
}

// Verdict derives the outcome from the counts, the same way the backfill migration does.
func (r *WorkflowRun) Verdict() RunOutcome {
	switch {
	case r.Errors > 0:
		return OutcomeErrors
	case r.Steps == 0:
		return OutcomeNoTrigger
	case r.NotifNone > 0 && r.NotifSent+r.NotifWouldSend == 0:
		return OutcomeNobodyNotified
	}
	return OutcomeClean
}

// RunFilter narrows the results list. Nil means any.
type RunFilter struct {
	BoardID  *int
	Outcome  RunOutcome
	TicketID *int
	From     *time.Time
	To       *time.Time
	// Before pages backwards: only runs that started before this instant.
	Before *time.Time
	Limit  int
}

// RunDetail is one run with its history events and the workflow as it is today.
type RunDetail struct {
	Run      WorkflowRun    `json:"run"`
	Events   []*TicketEvent `json:"events"`
	Workflow *Workflow      `json:"workflow,omitempty"`
}
