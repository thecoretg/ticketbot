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

func (s *server) processCompanyPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.JSON(http.StatusInternalServerError, util.ErrorJSON("invalid request body"))
		return
	}

	if w.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "company ID cannot be 0"})
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteCompany(w.ID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   fmt.Sprintf("couldn't delete company %v", err),
				"company": w.ID,
			})
			return
		}

		slog.Debug("company deleted", "id", w.ID)
		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processCompanyUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Debug("company was deleted externally", "id", w.ID)
				c.Status(http.StatusGone)
				return
			}

			slog.Error("deleting company", "companyID", w.ID)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err,
				"action": w.Action,
				"ticket": w.ID,
			})
			return
		}

		slog.Debug("company processed", "id", w.ID, "action", w.Action)
		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processCompanyUpdate(ctx context.Context, companyID int) error {
	cwc, err := s.cwClient.GetCompany(ctx, companyID, nil)
	if err != nil {
		return checkCWError("getting company via CW API", "company", err, companyID)
	}

	c := db.NewCompany(companyID, cwc.Name)
	if err := s.dbHandler.UpsertCompany(c); err != nil {
		return fmt.Errorf("processing company in db: %w", err)
	}

	return nil
}
