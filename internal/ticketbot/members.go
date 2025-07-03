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

func (s *server) processMemberPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("member ID cannot be 0"))
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteMember(w.ID); err != nil {
			c.Error(fmt.Errorf("deleting member %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processMemberUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Info("member was deleted externally", "id", w.ID)
				if err := s.dbHandler.DeleteMember(w.ID); err != nil {
					c.Error(fmt.Errorf("deleting member %d after external deletion: %w", w.ID, err))
					return
				}
				c.Status(http.StatusNoContent)
				return
			}

			c.Error(fmt.Errorf("processing member %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processMemberUpdate(ctx context.Context, memberID int) error {
	cwm, err := s.cwClient.GetMember(ctx, memberID, nil)
	if err != nil {
		return checkCWError("getting member via CW API", "member", err, memberID)
	}

	m := db.NewMember(memberID, cwm.Identifier, cwm.FirstName, cwm.LastName, cwm.PrimaryEmail, cwm.DefaultPhone)
	if err := s.dbHandler.UpsertMember(m); err != nil {
		return fmt.Errorf("processing contact in db: %w", err)
	}

	return nil
}
