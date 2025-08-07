package ticketbot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func (s *server) addBoardsGroup() {
	boards := s.ginEngine.Group("/boards", ErrorHandler(s.config.ExitOnError))
	boards.PUT("/:board_id", s.putBoard)
}

func (s *server) putBoard(c *gin.Context) {
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	storeBoard, err := s.dataStore.GetBoard(boardID)
	if err != nil {
		c.Error(fmt.Errorf("getting board: %w", err))
		return
	}

	if storeBoard == nil {
		c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board with id %d not found", boardID)))
		return
	}

	updatedBoard := &Board{}
	if err := c.ShouldBindJSON(updatedBoard); err != nil {
		c.Error(fmt.Errorf("unmarshaling board data: %w", err))
		return
	}

	if err := s.dataStore.UpsertBoard(updatedBoard); err != nil {
		c.Error(fmt.Errorf("upserting board: %w", err))
		return
	}

	c.JSON(http.StatusOK, updatedBoard)
}

func (s *server) ensureBoardInStore(cwData *cwData) (*Board, error) {
	board, err := s.dataStore.GetBoard(cwData.ticket.Board.ID)
	if err != nil {
		return nil, fmt.Errorf("getting board from storage: %w", err)
	}

	if board == nil {
		board, err = s.addBoard(cwData.ticket.Board.ID)
		if err != nil {
			return nil, fmt.Errorf("inserting board into store: %w", err)
		}
	}

	return board, nil
}

// addBoard adds Connecwise boards to the data store, with a default of
// notifications not enabled.
func (s *server) addBoard(boardID int) (*Board, error) {
	cwBoard, err := s.cwClient.GetBoard(boardID, nil)
	if err != nil {
		return nil, fmt.Errorf("getting board from connectwise: %w", err)
	}

	storeBoard := &Board{
		ID:            cwBoard.ID,
		Name:          cwBoard.Name,
		NotifyEnabled: false,
	}

	if err := s.dataStore.UpsertBoard(storeBoard); err != nil {
		return nil, fmt.Errorf("adding board to store: %w", err)
	}

	return storeBoard, nil
}
