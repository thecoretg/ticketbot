package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (s *Server) handleGetBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	board, err := s.Queries.GetBoard(c.Request.Context(), boardID)
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

func (s *Server) handleListBoards(c *gin.Context) {
	boards, err := s.Queries.ListBoards(c.Request.Context())
	if err != nil {
		c.Error(fmt.Errorf("listing boards: %w", err))
		return
	}

	if boards == nil {
		boards = []db.CwBoard{}
	}

	c.JSON(http.StatusOK, boards)
}

func (s *Server) handlePutBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	board, err := s.Queries.GetBoard(c.Request.Context(), boardID)
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

	updatedBoard, err := s.Queries.UpdateBoard(c.Request.Context(), db.UpdateBoardParams{
		ID:            board.ID,
		Name:          board.Name,
		NotifyEnabled: board.NotifyEnabled,
		WebexRoomID:   board.WebexRoomID,
	})

	c.JSON(http.StatusOK, updatedBoard)
}

func (s *Server) handleDeleteBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	err = s.Queries.DeleteBoard(c.Request.Context(), boardID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board %d not found", boardID)))
			return
		}
		c.Error(fmt.Errorf("deleting board: %w", err))
		return
	}
}

func (s *Server) ensureBoardInStore(ctx context.Context, cwData *cwData) (db.CwBoard, error) {
	board, err := s.Queries.GetBoard(ctx, cwData.ticket.Board.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("board not in store, attempting insert", "board_id", cwData.ticket.Board.ID)
			p := db.InsertBoardParams{
				ID:            cwData.ticket.Board.ID,
				Name:          cwData.ticket.Board.Name,
				NotifyEnabled: false,
				WebexRoomID:   nil,
			}
			board, err = s.Queries.InsertBoard(ctx, p)
			if err != nil {
				return db.CwBoard{}, fmt.Errorf("inserting board into db: %w", err)
			}
			slog.Debug("inserted board into store", "board_id", board.ID, "name", board.Name)
			return board, nil
		} else {
			return db.CwBoard{}, fmt.Errorf("getting board from storage: %w", err)
		}
	}

	slog.Debug("got existing board from store", "board_id", board.ID, "name", board.Name)
	return board, nil
}
