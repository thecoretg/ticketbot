package handlers

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/service/intake"
	"github.com/thecoretg/ticketbot/models"
)

// TicketbotHandler receives ConnectWise ticket callbacks. It only queues them: the intake workers
// do the fetch, diff, workflow run and notifications, with retries that survive a restart.
type TicketbotHandler struct {
	Intake *intake.Service
}

func NewTicketbotHandler(svc *intake.Service) *TicketbotHandler {
	return &TicketbotHandler{Intake: svc}
}

func (h *TicketbotHandler) ProcessTicket(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		badPayloadError(w, err)
		return
	}
	p := &psa.WebhookPayload{}
	if err := json.Unmarshal(body, p); err != nil {
		badPayloadError(w, err)
		return
	}

	switch action := models.IntakeAction(p.Action); action {
	case models.IntakeAdded, models.IntakeUpdated, models.IntakeDeleted:
		if _, err := h.Intake.Enqueue(r.Context(), p.ID, action, body); err != nil {
			// A 500 asks ConnectWise to retry the delivery, which is the right fallback when the
			// database itself is the problem.
			internalServerError(w, err)
			return
		}
	default:
		slog.Warn("unknown ticket webhook action", "action", p.Action, "ticket_id", p.ID)
	}

	resultJSON(w, "ticket payload received")
}
