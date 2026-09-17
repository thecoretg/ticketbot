package handlers

import (
	"context"
	"log/slog"
	"time"

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

// webhookRetryDelays paces retries after a failed intake. ConnectWise delivers a webhook once, so
// a transient ConnectWise or database error would otherwise lose the change until the next sync.
// Intake is idempotent: a retry diffs against whatever the previous attempt managed to store.
var webhookRetryDelays = []time.Duration{2 * time.Second, 10 * time.Second, 30 * time.Second}

func (h *TicketbotHandler) processTicket(ctx context.Context, id int, opts ticketbot.ProcessOpts) {
	for attempt := 0; ; attempt++ {
		err := h.Service.ProcessTicket(ctx, id, opts)
		if err == nil {
			return
		}
		if attempt >= len(webhookRetryDelays) {
			slog.Error("processing ticket webhook: giving up", "ticket_id", id, "attempts", attempt+1, "error", err.Error())
			return
		}

		delay := webhookRetryDelays[attempt]
		slog.Warn("processing ticket webhook failed; retrying", "ticket_id", id, "attempt", attempt+1, "retry_in", delay.String(), "error", err.Error())
		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return
		}
	}
}

func (h *TicketbotHandler) deleteTicket(ctx context.Context, id int) {
	if err := h.Service.SoftDeleteTicket(ctx, id, models.SourceWebhook); err != nil {
		slog.Error("soft deleting ticket from webhook", "ticket_id", id, "error", err.Error())
	}
}
