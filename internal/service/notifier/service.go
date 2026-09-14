package notifier

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Cfg           *models.Config
	WebexSvc      *webexsvc.Service
	Notifications repos.TicketNotificationRepository
	Forwards      repos.NotifierForwardRepository
	Pool          *pgxpool.Pool
	MessageSender repos.MessageSender
	CWCompanyID   string
}

type SvcParams struct {
	Cfg           *models.Config
	WebexSvc      *webexsvc.Service
	Notifications repos.TicketNotificationRepository
	Forwards      repos.NotifierForwardRepository
	Pool          *pgxpool.Pool
	MessageSender repos.MessageSender
	CWCompanyID   string
}

func New(p SvcParams) *Service {
	return &Service{
		Cfg:           p.Cfg,
		WebexSvc:      p.WebexSvc,
		Notifications: p.Notifications,
		Forwards:      p.Forwards,
		Pool:          p.Pool,
		MessageSender: p.MessageSender,
		CWCompanyID:   p.CWCompanyID,
	}
}
