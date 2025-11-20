package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrUserForwardNotFound = errors.New("forward rule not found")

type UserForward struct {
	ID            int        `json:"id"`
	UserEmail     string     `json:"user_email"`
	DestRoomID    int        `json:"dest_room_id"`
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Enabled       bool       `json:"enabled"`
	UserKeepsCopy bool       `json:"user_keeps_copy"`
	UpdatedOn     time.Time  `json:"updated_on"`
	AddedOn       time.Time  `json:"added_on"`
}

type UserForwardRepository interface {
	WithTx(tx pgx.Tx) UserForwardRepository
	ListAll(ctx context.Context) ([]UserForward, error)
	ListByEmail(ctx context.Context, email string) ([]UserForward, error)
	Get(ctx context.Context, id int) (UserForward, error)
	Insert(ctx context.Context, c UserForward) (UserForward, error)
	Delete(ctx context.Context, id int) error
}

var ErrNotifierNotFound = errors.New("notifier not found")

type Notifier struct {
	ID            int       `json:"id"`
	CwBoardID     int       `json:"cw_board_id"`
	WebexRoomID   int       `json:"webex_room_id"`
	NotifyEnabled bool      `json:"notify_enabled"`
	CreatedOn     time.Time `json:"created_on"`
}

type NotifierRepository interface {
	WithTx(tx pgx.Tx) NotifierRepository
	ListAll(ctx context.Context) ([]Notifier, error)
	ListByBoard(ctx context.Context, boardID int) ([]Notifier, error)
	ListByRoom(ctx context.Context, roomID int) ([]Notifier, error)
	Get(ctx context.Context, id int) (*Notifier, error)
	Exists(ctx context.Context, boardID, roomID int) (bool, error)
	Insert(ctx context.Context, n *Notifier) (*Notifier, error)
	Update(ctx context.Context, n *Notifier) (*Notifier, error)
	Delete(ctx context.Context, id int) error
}

var ErrNotificationNotFound = errors.New("notification not found")

type TicketNotification struct {
	ID           int       `json:"id"`
	NotifierID   *int      `json:"notifier_id"`
	TicketNoteID int       `json:"ticket_note_id"`
	WebexRoomID  *int      `json:"webex_room_id"`
	SentToEmail  *string   `json:"sent_to_email"`
	Sent         bool      `json:"sent"`
	Skipped      bool      `json:"skipped"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

type TicketNotificationRepository interface {
	WithTx(tx pgx.Tx) TicketNotificationRepository
	ListAll(ctx context.Context) ([]TicketNotification, error)
	ListByNoteID(ctx context.Context, noteID int) ([]TicketNotification, error)
	ExistsForNote(ctx context.Context, noteID int) (bool, error)
	Get(ctx context.Context, id int) (TicketNotification, error)
	Insert(ctx context.Context, n TicketNotification) (TicketNotification, error)
	Delete(ctx context.Context, id int) error
}
