package handlers

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/models"
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
		badPayloadError(c, err)
		return
	}
	id := w.ID
	action := w.Action

	ctx := context.WithoutCancel(c.Request.Context())
	switch action {
	case "added", "updated":
		opts := ticketbot.ProcessOpts{Source: models.SourceWebhook, WebhookMemberID: w.MemberID, RunRules: true}
		go h.processTicket(ctx, id, opts)
	case "deleted":
		go h.deleteTicket(ctx, id)
	default:
		slog.Warn("unknown ticket webhook action", "action", action, "ticket_id", id)
	}

	resultJSON(c, "ticket payload received")
}

func (h *TicketbotHandler) processTicket(ctx context.Context, id int, opts ticketbot.ProcessOpts) {
	if err := h.Service.ProcessTicket(ctx, id, opts); err != nil {
		slog.Error("processing ticket webhook", "ticket_id", id, "error", err.Error())
	}
}

func (h *TicketbotHandler) deleteTicket(ctx context.Context, id int) {
	if err := h.Service.SoftDeleteTicket(ctx, id, models.SourceWebhook); err != nil {
		slog.Error("soft deleting ticket from webhook", "ticket_id", id, "error", err.Error())
	}
}
