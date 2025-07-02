package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"tctg-automation/internal/ticketbot/db"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/util"
)

func (s *server) processContactPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.JSON(http.StatusInternalServerError, util.ErrorJSON("invalid request body"))
		return
	}

	if w.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "contact ID cannot be 0"})
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteContact(w.ID); err != nil {
			slog.Error("deleting contact", "id", w.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("couldn't delete contact %v", err),
				"contact": w.ID,
			})
			return
		}

		slog.Debug("contact deleted", "id", w.ID)
		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processContactUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Warn("contact was deleted externally", "id", w.ID)
				c.Status(http.StatusGone)
				return
			}

			slog.Error("processing contact", "id", w.ID, "action", w.Action, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err,
				"action": w.Action,
				"ticket": w.ID,
			})
			return
		}

		slog.Debug("contact processed", "id", w.ID, "action", w.Action)
		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processContactUpdate(ctx context.Context, contactID int) error {
	cwc, err := s.cwClient.GetContact(ctx, contactID, nil)
	if err != nil {
		return checkCWError("getting contact via CW API", "contact", err, contactID)
	}

	c := db.NewContact(contactID, cwc.FirstName, cwc.LastName, cwc.Company.ID)
	if err := s.dbHandler.UpsertContact(c); err != nil {
		return fmt.Errorf("processing contact in db: %w", err)
	}

	return nil
}
