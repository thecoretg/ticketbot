package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/util"
)

type ErrTicketWasDeleted struct {
	TicketID int
}

func (e ErrTicketWasDeleted) Error() string {
	return fmt.Sprintf("ticket %d was deleted by external factors", e.TicketID)
}

type ErrMaxRetries struct {
	TicketID  int
	Attempts  int
	LastError error
}

func (e ErrMaxRetries) Error() string {
	return fmt.Sprintf("max retries exceeded for ticket %d after %d attempts: %v", e.TicketID, e.Attempts, e.LastError)
}

func (s *server) processTicketPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.JSON(http.StatusInternalServerError, util.ErrorJSON("invalid request body"))
		return
	}

	if w.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ticket ID cannot be 0"})
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteTicket(w.ID); err != nil {
			slog.Error("deleting ticket", "id", w.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  fmt.Sprintf("couldn't delete ticket: %v", err),
				"ticket": w.ID,
			})
			return
		}

		slog.Info("ticket deleted", "id", w.ID)
		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processTicketUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrTicketWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Warn("ticket was deleted externally", "id", w.ID)
				c.Status(http.StatusGone)
				return
			}

			slog.Error("processing ticket", "id", w.ID, "action", w.Action, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err,
				"action": w.Action,
				"ticket": w.ID,
			})
			return
		}

		slog.Info("ticket processed", "id", w.ID, "action", w.Action)
		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processTicketUpdate(ctx context.Context, ticketID int) error {
	cwt, err := s.cwClient.GetTicket(ctx, ticketID, nil)
	if err != nil {
		return checkCWError("getting ticket info via CW API", err, ticketID)
	}

	n, err := s.getMostRecentNoteID(ctx, ticketID)
	if err != nil {
		return fmt.Errorf("getting most recent note: %w", err)
	}

	t := NewTicket(ticketID, cwt.Board.ID, cwt.Company.ID, cwt.Contact.ID, n, cwt.Owner.ID, cwt.Summary, cwt.Resources, cwt.Info.DateEntered, cwt.Info.LastUpdated, cwt.ClosedDate, cwt.ClosedFlag)
	if err := s.dbHandler.UpsertTicket(t); err != nil {
		return fmt.Errorf("processing ticket in db: %w", err)
	}

	return nil
}

func (s *server) getMostRecentNoteID(ctx context.Context, ticketID int) (int, error) {
	p := &connectwise.QueryParams{OrderBy: "_info/dateEntered desc"}
	n, err := s.cwClient.ListServiceTicketNotes(ctx, ticketID, p)
	if err != nil {
		return 0, checkCWError("listing ticket notes", err, ticketID)
	}

	for _, note := range n {
		if note.Text != "" {
			return note.ID, nil
		}
	}

	return 0, nil
}

// checks for specific errors to reduce repetitive connectwise error checking
func checkCWError(msg string, err error, ticketID int) error {
	var notFoundErr *connectwise.ErrNotFound

	switch {
	case errors.As(err, &notFoundErr):
		slog.Info("ticket was deleted externally", "id", ticketID)
		return ErrTicketWasDeleted{
			TicketID: ticketID,
		}
	default:
		return fmt.Errorf("%s: %w", msg, err)
	}
}
