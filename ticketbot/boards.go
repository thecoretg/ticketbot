package ticketbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"tctg-automation/db"
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

	storeBoard, err := s.queries.GetBoard(c.Request.Context(), boardID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board %d not found", boardID)))
		}
		c.Error(fmt.Errorf("getting board: %w", err))
		return
	}

	board := &db.Board{}
	if err := c.ShouldBindJSON(board); err != nil {
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

func (s *Server) ensureBoardInStore(ctx context.Context, cwData *cwData) (*Board, error) {
	board, err := s.queries.GetBoard(ctx, cwData.ticket.Board.ID)
	if err != nil {

		return nil, fmt.Errorf("getting board from storage: %w", err)
	}

	return board, nil
}

// addBoard adds Connecwise boards to the data store, with a default of
// notifications not enabled.
func (s *Server) addBoard(boardID int) (*Board, error) {
	cwBoard, err := s.cwClient.GetBoard(boardID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting board from connectwise: %w", err)
	}

	storeBoard := &Board{
		ID:            cwBoard.ID,
		Name:          cwBoard.Name,
		NotifyEnabled: false,
	}

	if err := s.queries.UpsertBoard(storeBoard); err != nil {
		return nil, fmt.Errorf("adding board to store: %w", err)
	}

	return storeBoard, nil
}
