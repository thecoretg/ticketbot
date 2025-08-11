package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/db"
	"log/slog"
	"net/http"
	"strconv"
)

func (s *Server) addBoardsGroup() {
	boards := s.ginEngine.Group("/boards", ErrorHandler(s.config.ExitOnError))
	boards.PUT("/:board_id", s.putBoard)
}

func (s *Server) putBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	board, err := s.queries.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board %d not found", boardID)))
		}
		c.Error(fmt.Errorf("getting board: %w", err))
		return
	}

	j := &board
	if err := c.ShouldBindJSON(j); err != nil {
		c.Error(fmt.Errorf("unmarshaling board data: %w", err))
		return
	}

	updatedBoard, err := s.queries.UpdateBoard(c.Request.Context(), db.UpdateBoardParams{
		ID:            board.ID,
		Name:          board.Name,
		NotifyEnabled: board.NotifyEnabled,
		WebexRoomID:   board.WebexRoomID,
	})

	c.JSON(http.StatusOK, updatedBoard)
}

func (s *Server) ensureBoardInStore(ctx context.Context, cwData *cwData) (db.Board, error) {
	board, err := s.queries.GetBoard(ctx, cwData.ticket.Board.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p := db.InsertBoardParams{
				ID:            cwData.ticket.Board.ID,
				Name:          cwData.ticket.Board.Name,
				NotifyEnabled: false,
				WebexRoomID:   nil,
			}
			board, err = s.queries.InsertBoard(ctx, p)
			if err != nil {
				return db.Board{}, fmt.Errorf("inserting board into db: %w", err)
			}
			slog.Debug("inserted board into store", "board_id", board.ID, "name", board.Name)
			return board, nil
		} else {
			return db.Board{}, fmt.Errorf("getting board from storage: %w", err)
		}
	}

	slog.Debug("got existing board from store", "board_id", board.ID, "name", board.Name)
	return board, nil
}
