package repos

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type MessageSender interface {
	GetMessage(id string, params map[string]string) (*webex.Message, error)
	GetAttachmentAction(messageID string) (*webex.AttachmentAction, error)
	PostMessage(message *webex.Message) (*webex.Message, error)
	ListRooms(params map[string]string) ([]webex.Room, error)
	ListPeople(email string) ([]webex.Person, error)
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
	Delete(ctx context.Context, id int) error
}

type NotifierRuleRepository interface {
	WithTx(tx pgx.Tx) NotifierRuleRepository
	ListAll(ctx context.Context) ([]*models.NotifierRule, error)
	ListAllFull(ctx context.Context) ([]*models.NotifierRuleFull, error)
	ListByBoard(ctx context.Context, boardID int) ([]*models.NotifierRule, error)
	ListByRoom(ctx context.Context, roomID int) ([]*models.NotifierRule, error)
	Get(ctx context.Context, id int) (*models.NotifierRule, error)
	Exists(ctx context.Context, id int) (bool, error)
	ExistsByBoardAndRecipient(ctx context.Context, boardID, roomID int) (bool, error)
	Insert(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error)
	Update(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error)
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
