package handler

import (
	"context"
	"fmt"
	"log/slog"

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

	ctx := context.WithoutCancel(c.Request.Context())
	switch action {
	case "added":
		go h.processTicket(ctx, id, true)
	case "updated":
		go h.processTicket(ctx, id, false)
	case "deleted":
		go h.deleteTicket(ctx, id)
	default:
		slog.Warn("unknown ticket webhook action", "action", action, "ticket_id", id)
	}

	c.JSON(200, gin.H{"result": "ticket webhook received"})
}

func (h *TicketbotHandler) processTicket(ctx context.Context, id int, isNew bool) {
	if err := h.Service.ProcessTicket(ctx, id, isNew); err != nil {
		slog.Error("processing ticket webhook", "ticket_id", id, "error", err)
	}
}

func (h *TicketbotHandler) deleteTicket(ctx context.Context, id int) {
	if err := h.Service.CW.DeleteTicket(ctx, id); err != nil {
		slog.Error("deleting ticket from webhook", "ticket_id", id, "error", err)
	}
}
