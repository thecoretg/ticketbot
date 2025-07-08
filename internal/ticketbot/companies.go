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
)

func (s *server) processCompanyPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("unmarshaling ConnectWise webhook payload: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("company ID cannot be 0"))
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteCompany(w.ID); err != nil {
			c.Error(fmt.Errorf("deleting company %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processCompanyUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Info("company was deleted externally", "id", w.ID)
				if err := s.dbHandler.DeleteCompany(w.ID); err != nil {
					c.Error(fmt.Errorf("deleting company %d after external deletion: %w", w.ID, err))
					return
				}
				c.Status(http.StatusNoContent)
				return
			}

			c.Error(fmt.Errorf("processing company %d: %w", w.ID, err))
			return
		}

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

func (s *server) ensureCompanyExists(companyID int, name string) error {
	c, err := s.dbHandler.GetCompany(companyID)
	if err != nil {
		return fmt.Errorf("querying db for company: %w", err)
	}

	if c == nil {
		n := db.NewCompany(companyID, name)
		if err := s.dbHandler.UpsertCompany(n); err != nil {
			return fmt.Errorf("inserting new company into db: %w", err)
		}
	}

	return nil
}
