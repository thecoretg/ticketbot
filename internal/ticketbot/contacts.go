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

func (s *server) processContactPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("contact ID cannot be 0"))
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteContact(w.ID); err != nil {
			c.Error(fmt.Errorf("deleting contact %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processContactUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Info("contact was deleted externally", "id", w.ID)
				if err := s.dbHandler.DeleteContact(w.ID); err != nil {
					c.Error(fmt.Errorf("deleting contact %d after external deletion: %w", w.ID, err))
					return
				}
				c.Status(http.StatusNoContent)
				return
			}

			c.Error(fmt.Errorf("processing contact %d: %w", w.ID, err))
			return
		}

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

func (s *server) ensureContactExists(ctx context.Context, contactID int) error {
	c, err := s.dbHandler.GetContact(contactID)
	if err != nil {
		return fmt.Errorf("querying db for contact: %w", err)
	}

	if c == nil {
		r, err := s.cwClient.GetContact(ctx, contactID, nil)
		if err != nil {
			return checkCWError("getting contact", "contact", err, contactID)
		}

		if r.Company.ID != 0 {
			if err := s.ensureCompanyExists(r.Company.ID, r.Company.Name); err != nil {
				return fmt.Errorf("ensuring company exists for contact: %w", err)
			}
		}

		n := db.NewContact(contactID, r.FirstName, r.LastName, r.Company.ID)
		if err := s.dbHandler.UpsertContact(n); err != nil {
			return fmt.Errorf("inserting new contact into db: %w", err)
		}
	}

	return nil
}
