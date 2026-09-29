package models

import (
	"encoding/json"
	"errors"
	"time"
)

var (
	ErrIntakeEmpty    = errors.New("no webhook ready for processing")
	ErrIntakeNotFound = errors.New("queued webhook not found or not failed")
)

// IntakeStatus is the lifecycle of one queued webhook.
type IntakeStatus string

const (
	IntakePending    IntakeStatus = "pending"    // waiting for the worker, or for its next attempt
	IntakeProcessing IntakeStatus = "processing" // claimed by the worker
	IntakeDone       IntakeStatus = "done"       // processed; purged after the log retention period
	IntakeFailed     IntakeStatus = "failed"     // every attempt failed; waits for retry or discard
	IntakeDiscarded  IntakeStatus = "discarded"  // an admin dropped it; purged like done
)

// IntakeAction is the ConnectWise callback action a queued webhook carries, or IntakeCatchup for
// a row the missed-webhook check queued because ConnectWise changed a ticket without calling back.
type IntakeAction string

const (
	IntakeAdded   IntakeAction = "added"
	IntakeUpdated IntakeAction = "updated"
	IntakeDeleted IntakeAction = "deleted"
	IntakeCatchup IntakeAction = "catchup"
)

// WebhookIntake is one ConnectWise ticket webhook, persisted on arrival so a restart or a transient
// failure never loses it.
type WebhookIntake struct {
	ID            int64           `json:"id"`
	TicketID      int             `json:"ticket_id"`
	Action        IntakeAction    `json:"action"`
	Payload       json.RawMessage `json:"payload"`
	Status        IntakeStatus    `json:"status"`
	Attempts      int             `json:"attempts"`
	NextAttemptAt time.Time       `json:"next_attempt_at"`
	LastError     *string         `json:"last_error,omitempty"`
	ReceivedAt    time.Time       `json:"received_at"`
	FinishedAt    *time.Time      `json:"finished_at,omitempty"`
}

// IntakeHourCount is one bar of the webhooks-per-hour chart.
type IntakeHourCount struct {
	Hour  time.Time `json:"hour"`
	Count int64     `json:"count"`
}

// IntakeStats summarises the queue for the dashboard.
type IntakeStats struct {
	Counts map[IntakeStatus]int64 `json:"counts"`
	// LastReceivedAt is the last real webhook; catch-up rows do not count.
	LastReceivedAt *time.Time `json:"last_received_at,omitempty"`
	// CatchupLast7Days is how many rows the missed-webhook check queued in the last seven days
	// (bounded by the intake retention). Catchup is that check's state, when it has run.
	CatchupLast7Days int64         `json:"catchup_last_7_days"`
	Catchup          *CatchupState `json:"catchup,omitempty"`
}

// CatchupState is the missed-webhook check's single state row. CheckedThrough is the watermark:
// the start of the last run that succeeded; the next run asks ConnectWise for tickets changed
// since then. A failed run records LastRunAt and LastError and leaves the watermark alone.
type CatchupState struct {
	CheckedThrough *time.Time `json:"checked_through,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	LastQueued     int        `json:"last_queued"`
	LastError      *string    `json:"last_error,omitempty"`
}
