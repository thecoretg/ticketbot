package models

import (
	"errors"
	"time"
)

var ErrUserForwardNotFound = errors.New("forward rule not found")

type NotifierForward struct {
	ID            int        `json:"id"`
	SourceID      int        `json:"user_email"`
	DestID        int        `json:"dest_email"`
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Enabled       bool       `json:"enabled"`
	UserKeepsCopy bool       `json:"user_keeps_copy"`
	CreatedOn     time.Time  `json:"added_on"`
	UpdatedOn     time.Time  `json:"updated_on"`
}

type NotifierForwardFull struct {
	ID              int        `json:"id"`
	Enabled         bool       `json:"enabled"`
	UserKeepsCopy   bool       `json:"user_keeps_copy"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	SourceID        int        `json:"source_id"`
	SourceName      string     `json:"source_name"`
	SourceType      string     `json:"source_type"`
	DestinationID   int        `json:"destination_id"`
	DestinationName string     `json:"destination_name"`
	DestinationType string     `json:"destination_type"`
}

var ErrNotifierNotFound = errors.New("notifier not found")

type NotifierRule struct {
	ID               int       `json:"id"`
	CwBoardID        int       `json:"cw_board_id"`
	WebexRecipientID int       `json:"webex_room_id"`
	NotifyEnabled    bool      `json:"notify_enabled"`
	CreatedOn        time.Time `json:"created_on"`
}

type NotifierRuleFull struct {
	ID            int    `json:"id"`
	Enabled       bool   `json:"enabled"`
	BoardID       int    `json:"board_id"`
	BoardName     string `json:"board_name"`
	RecipientID   int    `json:"recipient_id"`
	RecipientName string `json:"recipient_name"`
	RecipientType string `json:"recipient_type"`
}

var ErrNotificationNotFound = errors.New("notification not found")

type TicketNotification struct {
	ID              int       `json:"id"`
	TicketID        int       `json:"ticket_id"`
	TicketNoteID    *int      `json:"ticket_note_id"`
	RecipientID     *int      `json:"webex_room_id"`
	ForwardedFromID *int      `json:"forwarded_from_id"`
	Sent            bool      `json:"sent"`
	Skipped         bool      `json:"skipped"`
	CreatedOn       time.Time `json:"created_on"`
	UpdatedOn       time.Time `json:"updated_on"`
}
