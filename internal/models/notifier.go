package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type MessageSender interface {
	GetMessage(id string, params map[string]string) (*webex.Message, error)
	GetAttachmentAction(messageID string) (*webex.AttachmentAction, error)
	PostMessage(message *webex.Message) (*webex.Message, error)
	ListRooms(params map[string]string) ([]webex.Room, error)
	ListPeople(email string) ([]webex.Person, error)
}

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

type NotifierForwardRepository interface {
	WithTx(tx pgx.Tx) NotifierForwardRepository
	ListAll(ctx context.Context) ([]*NotifierForward, error)
	ListAllActive(ctx context.Context) ([]*NotifierForward, error)
	ListAllInactive(ctx context.Context) ([]*NotifierForward, error)
	ListAllFull(ctx context.Context) ([]*NotifierForwardFull, error)
	ListBySourceRoomID(ctx context.Context, id int) ([]*NotifierForward, error)
	ListActiveBySourceRoomID(ctx context.Context, id int) ([]*NotifierForward, error)
	Get(ctx context.Context, id int) (*NotifierForward, error)
	Exists(ctx context.Context, id int) (bool, error)
	Insert(ctx context.Context, c *NotifierForward) (*NotifierForward, error)
	Delete(ctx context.Context, id int) error
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

type NotifierRuleRepository interface {
	WithTx(tx pgx.Tx) NotifierRuleRepository
	ListAll(ctx context.Context) ([]*NotifierRule, error)
	ListAllFull(ctx context.Context) ([]*NotifierRuleFull, error)
	ListByBoard(ctx context.Context, boardID int) ([]*NotifierRule, error)
	ListByRoom(ctx context.Context, roomID int) ([]*NotifierRule, error)
	Get(ctx context.Context, id int) (*NotifierRule, error)
	Exists(ctx context.Context, id int) (bool, error)
	ExistsByBoardAndRecipient(ctx context.Context, boardID, roomID int) (bool, error)
	Insert(ctx context.Context, n *NotifierRule) (*NotifierRule, error)
	Update(ctx context.Context, n *NotifierRule) (*NotifierRule, error)
	Delete(ctx context.Context, id int) error
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

type TicketNotificationRepository interface {
	WithTx(tx pgx.Tx) TicketNotificationRepository
	ListAll(ctx context.Context) ([]*TicketNotification, error)
	ListByNoteID(ctx context.Context, noteID int) ([]*TicketNotification, error)
	ExistsForTicket(ctx context.Context, ticketID int) (bool, error)
	ExistsForNote(ctx context.Context, noteID int) (bool, error)
	Get(ctx context.Context, id int) (*TicketNotification, error)
	Insert(ctx context.Context, n *TicketNotification) (*TicketNotification, error)
	Delete(ctx context.Context, id int) error
}
