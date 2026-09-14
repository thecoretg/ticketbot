package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/models"
)

type MessageSender interface {
	GetMessage(ctx context.Context, id string, params map[string]string) (*webex.Message, error)
	GetAttachmentAction(ctx context.Context, messageID string) (*webex.AttachmentAction, error)
	PostMessage(ctx context.Context, message *webex.Message) (*webex.Message, error)
	ListRooms(ctx context.Context, params map[string]string) ([]webex.Room, error)
	ListPeople(ctx context.Context, email string) ([]webex.Person, error)
}

type NotifierForwardRepository interface {
	WithTx(tx pgx.Tx) NotifierForwardRepository
	ListAll(ctx context.Context) ([]*models.NotifierForward, error)
	ListAllActive(ctx context.Context) ([]*models.NotifierForwardFull, error)
	ListAllNotExpired(ctx context.Context) ([]*models.NotifierForwardFull, error)
	ListAllInactive(ctx context.Context) ([]*models.NotifierForwardFull, error)
	ListAllFull(ctx context.Context) ([]*models.NotifierForwardFull, error)
	ListBySourceRoomID(ctx context.Context, id int) ([]*models.NotifierForward, error)
	ListActiveBySourceRoomID(ctx context.Context, id int) ([]*models.NotifierForwardFull, error)
	Get(ctx context.Context, id int) (*models.NotifierForward, error)
	Exists(ctx context.Context, id int) (bool, error)
	Insert(ctx context.Context, c *models.NotifierForward) (*models.NotifierForward, error)
	Update(ctx context.Context, c *models.NotifierForward) (*models.NotifierForward, error)
	Delete(ctx context.Context, id int) error
}

type TicketNotificationRepository interface {
	WithTx(tx pgx.Tx) TicketNotificationRepository
	ListAll(ctx context.Context) ([]*models.TicketNotification, error)
	ListByNoteID(ctx context.Context, noteID int) ([]*models.TicketNotification, error)
	ExistsForTicket(ctx context.Context, ticketID int) (bool, error)
	ExistsForNote(ctx context.Context, noteID int) (bool, error)
	Get(ctx context.Context, id int) (*models.TicketNotification, error)
	Insert(ctx context.Context, n *models.TicketNotification) (*models.TicketNotification, error)
	Delete(ctx context.Context, id int) error
}
