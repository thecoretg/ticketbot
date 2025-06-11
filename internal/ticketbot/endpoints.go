package ticketbot

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log/slog"
	"net/http"
	"strconv"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/util"
)

func (s *Server) listBoardsEndpoint(c *gin.Context) {
	b := s.Boards
	if len(b) == 0 {
		b = []boardSetting{}
	}

	c.JSON(http.StatusOK, gin.H{"boards": b})
}

func (s *Server) addOrUpdateBoardEndpoint(c *gin.Context) {
	b := &boardSetting{}
	if err := c.ShouldBindJSON(b); err != nil {
		util.ErrorJSON(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := s.addBoardSetting(b); err != nil {
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to add or update board setting")
		return
	}

	if err := s.refreshBoards(); err != nil {
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to refresh boards")
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Board setting added or updated successfully"})
}

func (s *Server) deleteBoardEndpoint(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.Atoi(boardIDStr)
	if err != nil {
		util.ErrorJSON(c, http.StatusBadRequest, "board_id must be a valid integer")
		return
	}

	if err := s.deleteBoardSetting(boardID); err != nil {
		util.ErrorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	if err := s.refreshBoards(); err != nil {
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to refresh boards")
		return
	}

	c.Status(http.StatusNoContent)
}

func (s *Server) handleTicketEndpoint(c *gin.Context) {
	// Parse webhook payload
	w := &connectwise.WebhookPayload{}
	if err := c.ShouldBindJSON(w); err != nil {
		util.ErrorJSON(c, http.StatusInternalServerError, "invalid request body")
		return
	}

	// Validate action and ID
	if !validAction(w.Action) {
		slog.Debug("invalid action received", "action", w.Action, "ticketId", w.ID)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("action '%s' is not of 'added' or 'updated'", w.Action)})
		return
	}
	if w.ID == 0 {
		slog.Error("ticket ID is required", "action", w.Action)
		util.ErrorJSON(c, http.StatusBadRequest, "ticket ID is required")
		return
	}

	// Fetch ticket and validate board
	ticket, err := s.cwClient.GetTicket(c.Request.Context(), w.ID, nil)
	if err != nil {
		slog.Error("failed to get ticket", "ticketId", w.ID, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to get ticket")
		return
	}

	bs := s.ticketInEnabledBoard(ticket)
	if bs == nil {
		slog.Debug("ticket not in enabled board", "ticketId", ticket.ID, "boardId", ticket.Board.ID)
		c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("ticket %d received, but board %d is not an enabled board for notifications", ticket.ID, ticket.Board.ID)})
		return
	}

	p := &connectwise.QueryParams{OrderBy: "_info/dateEntered desc"}
	notes, err := s.cwClient.ListServiceTicketNotes(c.Request.Context(), ticket.ID, p)
	if err != nil {
		slog.Error("error receiving ticket notes", "ticketId", ticket.ID, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "error receiving ticket notes")
		return
	}

	switch w.Action {
	case "added":
		s.handleNewTicket(c, ticket, notes, bs)
	case "updated":
		s.handleUpdatedTicket(c, ticket, notes, bs, w)
	}
}

func validAction(action string) bool {
	return action == "added" || action == "updated"
}
