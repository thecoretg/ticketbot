package handler

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
)

type TicketbotHandler struct {
	Service *ticketbot.Service
}

func NewTicketbotHandler(svc *ticketbot.Service) *TicketbotHandler {
	return &TicketbotHandler{Service: svc}
}

func (h *TicketbotHandler) ProcessTicket(c *gin.Context) {
	w := &psa.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("bad json payload: %w", err))
		return
	}
	id := w.ID
	action := w.Action

	ctx := c.Request.Context()
	switch action {
	case "added":
		if err := h.Service.ProcessTicket(ctx, id, true); err != nil {
			c.Error(err)
			return
		}
	case "updated":
		if err := h.Service.ProcessTicket(ctx, id, false); err != nil {
			c.Error(err)
			return
		}
	case "deleted":
		if err := h.Service.CW.DeleteTicket(ctx, id); err != nil {
			c.Error(err)
			return
		}
	}
}
