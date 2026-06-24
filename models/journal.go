package models

import (
	"errors"
	"time"
)

var ErrTicketJournalNotFound = errors.New("ticket journal not found")

// Journal event statuses, used by the frontend to colour timeline lines.
const (
	JournalOK    = "ok"
	JournalError = "error"
	JournalSkip  = "skip"
	JournalInfo  = "info"
)

// Run triggers.
const (
	TriggerNew     = "New ticket"
	TriggerUpdated = "Ticket updated"
)

// Run outcomes.
const (
	OutcomeCompleted   = "Completed"
	OutcomeWithErrors  = "Completed with errors"
	OutcomeSimulated   = "Simulated" // nothing was actually done, but a simulation-mode entity reported what it would do
	OutcomeNothingToDo = "Nothing to do"
)

// JournalEvent is a single human-readable line in a ticket run's timeline.
type JournalEvent struct {
	Text      string   `json:"text"`
	Status    string   `json:"status"`              // ok | error | skip | info
	Simulated bool     `json:"simulated,omitempty"` // true when produced by a simulation-mode workflow/rule/forward ("Would ...")
	Tags      []string `json:"tags,omitempty"`      // optional colored badges rendered after the text (e.g. note flags)
	Workflow  string   `json:"workflow,omitempty"`  // name of the matched workflow this event belongs to (match line + its actions); empty for non-workflow events
}

// TicketRun is one pass of the ticketbot pipeline over a ticket, recorded as a
// friendly timeline entry rather than raw log lines.
type TicketRun struct {
	Time        time.Time      `json:"time"`
	Trigger     string         `json:"trigger"`                  // "New ticket" | "Ticket updated"
	Events      []JournalEvent `json:"events"`
	Outcome     string         `json:"outcome"`
	HadError    bool           `json:"had_error"`
	WorkflowRan bool           `json:"workflow_ran,omitempty"`   // true when the workflow pipeline executed (matched or not)
}

// TicketJournal is the per-ticket audit record. The denormalized snapshot columns
// (name fields) drive the overview table; Runs is the appended lifecycle timeline.
type TicketJournal struct {
	TicketID      int         `json:"ticket_id"`
	Summary       string      `json:"summary"`
	BoardName     string      `json:"board_name"`
	CompanyName   string      `json:"company_name"`
	ContactName   string      `json:"contact_name"`
	StatusName    string      `json:"status_name"`
	OwnerName     string      `json:"owner_name"`
	TypeName      string      `json:"type_name"`
	SubtypeName   string      `json:"subtype_name"`
	ItemName      string      `json:"item_name"`
	ResourceNames string      `json:"resource_names"`            // sorted comma-joined resource member names; stored in DB
	LastTrigger   string      `json:"last_trigger"`
	LastOutcome   string      `json:"last_outcome"`
	HadError      bool        `json:"had_error"`
	FirstSeen     time.Time   `json:"first_seen"`
	LastRun       time.Time   `json:"last_run"`
	Runs          []TicketRun `json:"runs,omitempty"`            // omitted in the overview list
	PsaURL        string      `json:"psa_url,omitempty"`         // computed on read; not stored in DB
}
