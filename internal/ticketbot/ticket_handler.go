package ticketbot

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"tctg-automation/internal/ticketbot/types"
	"tctg-automation/pkg/connectwise"
	"time"
)

func (s *server) addTicketGroup(r *gin.Engine) {
	tickets := r.Group("/hooks")
	cw := tickets.Group("/cw", requireValidCWSignature(), ErrorHandler(s.exitOnError))
	cw.POST("/tickets", s.handleTickets)
}

func (s *server) handleTickets(c *gin.Context) {
	w := connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling connectwise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("ticket ID cannot be 0"))
		return
	}

	slog.Info("received ticket webhook", "id", w.ID, "action", w.Action)
	if w.Action == "added" || w.Action == "updated" {
		ticket, err := s.cwClient.GetTicket(w.ID, nil)
		if err != nil {
			c.Error(fmt.Errorf("getting ticket from connectwise: %w", err))
			return
		}

		if err := s.addOrUpdateTicket(ticket); err != nil {
			c.Error(fmt.Errorf("adding or updating the ticket into data storage: %w", err))
			return
		}

		c.Status(http.StatusNoContent)
	}
}

func (s *server) addOrUpdateTicket(cwTicket *connectwise.Ticket) error {

	ticket := &types.Ticket{
		ID:      cwTicket.ID,
		Summary: cwTicket.Summary,
		TimeDetails: types.TimeDetails{
			UpdatedAt: time.Now(),
		},
	}

	if err := s.dataStore.UpsertTicket(ticket); err != nil {
		return err
	}

	return nil
}
