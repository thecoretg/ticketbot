package ticketbot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
)

func (s *server) addTicketsGroup(r *gin.Engine) {
	tickets := r.Group("/tickets")
	tickets.GET("", s.getTickets)
}

func (s *server) addBoardsGroup(r *gin.Engine) {
	boards := r.Group("/boards")
	boards.GET("", s.getBoards)
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
