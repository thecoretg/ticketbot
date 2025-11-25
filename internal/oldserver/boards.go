package oldserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) handleGetBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	board, err := cl.Queries.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board %d not found", boardID)))
			return
		}
		c.Error(fmt.Errorf("getting board: %w", err))
		return
	}

	c.JSON(http.StatusOK, board)
}

func (cl *Client) handleListBoards(c *gin.Context) {
	boards, err := cl.Queries.ListBoards(c.Request.Context())
	if err != nil {
		c.Error(fmt.Errorf("listing boards: %w", err))
		return
	}

	if boards == nil {
		boards = []db.CwBoard{}
	}

	c.JSON(http.StatusOK, boards)
}

func (cl *Client) ensureBoardInStore(ctx context.Context, q *db.Queries, boardID int) (db.CwBoard, error) {
	board, err := q.GetBoard(ctx, boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cwBoard, err := cl.CWClient.GetBoard(boardID, nil)
			if err != nil {
				return db.CwBoard{}, fmt.Errorf("getting board from cw: %w", err)
			}
			p := db.UpsertBoardParams{
				ID:   cwBoard.ID,
				Name: cwBoard.Name,
			}

			board, err = q.UpsertBoard(ctx, p)
			if err != nil {
				return db.CwBoard{}, fmt.Errorf("inserting board into db: %w", err)
			}
		} else {
			return db.CwBoard{}, fmt.Errorf("getting board from storage: %w", err)
		}
	}

	return board, nil
}
