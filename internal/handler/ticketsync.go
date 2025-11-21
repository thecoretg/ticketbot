package handler

import (
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
)

type TicketSyncHandler struct {
	CWService        *cwsvc.Service
	TicketbotService *notifier.Service
}
