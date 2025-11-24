package notifier

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Cfg              models.Config
	Rooms            models.WebexRoomRepository
	Notifiers        models.NotifierRepository
	Notifications    models.TicketNotificationRepository
	Forwards         models.UserForwardRepository
	Pool             *pgxpool.Pool
	MessageSender    models.MessageSender
	CWCompanyID      string
	MaxMessageLength int
}

type Repos struct {
	Rooms         models.WebexRoomRepository
	Notifiers     models.NotifierRepository
	Notifications models.TicketNotificationRepository
	Forwards      models.UserForwardRepository
}

func New(cfg models.Config, r Repos, ms models.MessageSender, cwCompanyID string, maxLen int) *Service {
	return &Service{
		Cfg:              cfg,
		Rooms:            r.Rooms,
		Notifiers:        r.Notifiers,
		Notifications:    r.Notifications,
		Forwards:         r.Forwards,
		MessageSender:    ms,
		CWCompanyID:      cwCompanyID,
		MaxMessageLength: maxLen,
	}
}
