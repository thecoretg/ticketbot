package ticketbot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
)

func (s *server) addTicketsGroup() {
	tickets := s.ginEngine.Group("/tickets", ErrorHandler(s.config.ExitOnError))
	tickets.GET("", s.getTickets)
}

func (s *server) addBoardsGroup() {
	boards := s.ginEngine.Group("/boards", ErrorHandler(s.config.ExitOnError))
	boards.GET("", s.getBoards)
	boards.GET(":board_id", s.getBoard)
	boards.PUT(":board_id", s.putBoard)
}

func (s *server) getTickets(c *gin.Context) {
	slog.Debug("get tickets called")
	tickets, err := s.dataStore.ListTickets()
	if err != nil {
		c.Error(fmt.Errorf("getting list of tickets: %w", err))
		return
	}
	slog.Debug("got tickets", "total_tickets", len(tickets))

	c.JSON(http.StatusOK, tickets)
}

func (s *server) getBoard(c *gin.Context) {
	slog.Debug("get board called")
	boardID, err := strconv.Atoi(c.Param("board_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("board id must be a valid integer"))
		return
	}

	board, err := s.dataStore.GetBoard(boardID)
	if err != nil {
		c.Error(fmt.Errorf("getting board: %w", err))
		return
	}

	if board == nil {
		c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("board with id %d not found", boardID)))
		return
	}

	c.JSON(http.StatusOK, board)
}

func (s *server) getBoards(c *gin.Context) {
	slog.Debug("get boards calleds")
	boards, err := s.dataStore.ListBoards()
	if err != nil {
		c.Error(fmt.Errorf("getting list of boards: %w", err))
		return
	}
	slog.Debug("got boards", "total_boards", len(boards))

	c.JSON(http.StatusOK, boards)
}

func (s *server) putBoard(c *gin.Context) {
	slog.Debug("put board called")
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
