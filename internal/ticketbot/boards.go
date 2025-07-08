package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"tctg-automation/internal/ticketbot/db"
	"tctg-automation/pkg/connectwise"
)

func (s *server) addBoardsGroup(r *gin.Engine) {
	bg := r.Group("/boards", ErrorHandler(s.exitOnError))
	bg.GET("/", s.listBoards)
	bg.GET("/:board_id", s.getBoard)
	bg.PUT("/:board_id/notify", s.updateBoardNotifySettings)
}

func (s *server) listBoards(c *gin.Context) {
	b, err := s.dbHandler.ListBoards()
	if err != nil {
		c.Error(fmt.Errorf("listing boards: %w", err))
		return
	}

	c.IndentedJSON(http.StatusOK, b)
}

func (s *server) updateBoardNotifySettings(c *gin.Context) {
	b := &db.BoardNotifySettings{}
	if err := c.ShouldBindJSON(b); err != nil {
		c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	boardIDStr := c.Param("board_id")
	boardID, err := strconv.Atoi(boardIDStr)
	if err != nil {
		c.Error(errors.New("board ID must be a valid integer"))
		return
	}

	if boardID == 0 {
		c.Error(errors.New("board ID cannot be 0"))
		return
	}

	if err := s.dbHandler.UpdateBoardNotifySettings(boardID, b.NotifyEnabled, b.WebexSpace); err != nil {
		c.Error(fmt.Errorf("updating board notify settings: %w", err))
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *server) getBoard(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.Atoi(boardIDStr)
	if err != nil {
		c.Error(errors.New("board ID must be a valid integer"))
		return
	}

	if boardID == 0 {
		c.Error(errors.New("board ID cannot be 0"))
		return
	}

	b, err := s.dbHandler.GetBoard(boardID)
	if err != nil {
		c.Error(err)
		return
	}

	c.IndentedJSON(http.StatusOK, b)
}

func (s *server) processBoardsPayload(c *gin.Context) {
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		c.Error(fmt.Errorf("invalid request body: %w", err))
		return
	}

	if w.ID == 0 {
		c.Error(errors.New("board ID cannot be 0"))
		return
	}
	switch w.Action {
	case "deleted":
		if err := s.dbHandler.DeleteBoard(w.ID); err != nil {
			c.Error(fmt.Errorf("deleting board %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	default:
		if err := s.processBoardUpdate(c.Request.Context(), w.ID); err != nil {
			var deletedErr ErrWasDeleted
			if errors.As(err, &deletedErr) {
				slog.Info("board was deleted externally", "id", w.ID)
				if err := s.dbHandler.DeleteBoard(w.ID); err != nil {
					c.Error(fmt.Errorf("deleting board %d after external deletion: %w", w.ID, err))
					return
				}
				c.Status(http.StatusNoContent)
				return
			}

			c.Error(fmt.Errorf("processing board %d: %w", w.ID, err))
			return
		}

		c.Status(http.StatusNoContent)
		return
	}
}

func (s *server) processBoardUpdate(ctx context.Context, boardID int) error {
	cwb, err := s.cwClient.GetBoard(ctx, boardID, nil)
	if err != nil {
		return checkCWError("getting board via CW API", "board", err, boardID)
	}

	c := db.NewBoard(boardID, cwb.Name)
	if err := s.dbHandler.UpsertBoard(c); err != nil {
		return fmt.Errorf("processing board in db: %w", err)
	}

	return nil
}

func (s *server) ensureBoardExists(boardID int, name string) error {
	b, err := s.dbHandler.GetBoard(boardID)
	if err != nil {
		return fmt.Errorf("querying db for board: %w", err)
	}

	if b == nil {
		n := db.NewBoard(boardID, name)
		if err := s.dbHandler.UpsertBoard(n); err != nil {
			return fmt.Errorf("inserting new board into db: %w", err)
		}
	}

	return nil
}

func (s *server) ensureStatusExists(ctx context.Context, statusID, boardID int, boardName string) error {
	st, err := s.dbHandler.GetStatus(statusID)
	if err != nil {
		return fmt.Errorf("querying db for status: %w", err)
	}

	if st == nil {

		if boardID == 0 {
			if err := s.ensureBoardExists(boardID, boardName); err != nil {
				return fmt.Errorf("ensuring board exists for status: %w", err)
			}
		}

		r, err := s.cwClient.GetBoardStatus(ctx, boardID, statusID, nil)
		if err != nil {
			return checkCWError("getting status", "status", err, statusID)
		}

		n := db.NewStatus(statusID, boardID, r.Name, r.ClosedStatus, !r.Inactive)
		if err := s.dbHandler.UpsertStatus(n); err != nil {
			return fmt.Errorf("inserting new status into db: %w", err)
		}
	}

	return nil
}
