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

func (s *server) processMemberPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.JSON(http.StatusInternalServerError, util.ErrorJSON("invalid request body"))
		return
	}

	if w.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "member ID cannot be 0"})
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteMember(w.ID); err != nil {
			slog.Error("deleting member", "id", w.ID, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  fmt.Sprintf("couldn't delete member %v", err),
				"member": w.ID,
			})
			return
		}

		slog.Debug("member deleted", "id", w.ID)
		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processMemberUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Debug("member was deleted externally", "id", w.ID)
				c.Status(http.StatusGone)
				return
			}

			slog.Error("processing member", "id", w.ID, "action", w.Action, "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":  err,
				"action": w.Action,
				"ticket": w.ID,
			})
			return
		}

		slog.Debug("member processed", "id", w.ID, "action", w.Action)
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
