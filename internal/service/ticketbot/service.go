package ticketbot

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
)

type Service struct {
	Rooms            models.WebexRoomRepository
	Notifiers        models.NotifierRepository
	Notifications    models.TicketNotificationRepository
	Forwards         models.UserForwardRepository
	Pool             *pgxpool.Pool
	WebexClient      *webex.Client
	CWCompanyID      string
	MaxMessageLength int
}

type Repos struct {
	Rooms         models.WebexRoomRepository
	Notifiers     models.NotifierRepository
	Notifications models.TicketNotificationRepository
	Forwards      models.UserForwardRepository
}

func New(r Repos, wc *webex.Client, cwCompanyID string, max int) *Service {
	return &Service{
		Rooms:            r.Rooms,
		Notifiers:        r.Notifiers,
		Notifications:    r.Notifications,
		Forwards:         r.Forwards,
		WebexClient:      wc,
		CWCompanyID:      cwCompanyID,
		MaxMessageLength: max,
	}
}
