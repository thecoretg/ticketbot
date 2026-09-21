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

// IntakeAction is the ConnectWise callback action a queued webhook carries.
type IntakeAction string

const (
	IntakeAdded   IntakeAction = "added"
	IntakeUpdated IntakeAction = "updated"
	IntakeDeleted IntakeAction = "deleted"
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
	Counts         map[IntakeStatus]int64 `json:"counts"`
	LastReceivedAt *time.Time             `json:"last_received_at,omitempty"`
}
