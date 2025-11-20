package handler

import (
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
)

type TicketSyncHandler struct {
	CWService        *cwsvc.Service
	TicketbotService *ticketbot.Service
}
