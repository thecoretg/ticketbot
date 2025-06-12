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

func (s *server) listBoardsEndpoint(c *gin.Context) {
	boards, err := getAllBoards(s.db)
	if err != nil {
		slog.Error("failed to get boards from database", "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to get boards")
		return
	}

	c.JSON(http.StatusOK, boards)
}

func (s *server) addOrUpdateBoardEndpoint(c *gin.Context) {
	b := &boardSetting{}
	if err := c.ShouldBindJSON(b); err != nil {
		slog.Error("failed to bind JSON to boardSetting", "error", err)
		util.ErrorJSON(c, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedBoard, err := addOrUpdateBoard(s.db, b)
	if err != nil {
		slog.Error("failed to add or update board setting", "boardId", b.BoardID, "boardName", b.BoardName, "webexRoomID", b.WebexRoomID, "enabled", b.Enabled, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to add or update board setting")
		return
	}

	slog.Info("board setting added or updated", "boardId", b.BoardID, "boardName", b.BoardName, "webexRoomID", b.WebexRoomID, "enabled", b.Enabled)
	c.JSON(http.StatusOK, updatedBoard)
}

func (s *server) deleteBoardEndpoint(c *gin.Context) {
	boardIDStr := c.Param("board_id")
	boardID, err := strconv.Atoi(boardIDStr)
	if err != nil {
		slog.Error("invalid board_id parameter", "board_id", boardIDStr, "error", err)
		util.ErrorJSON(c, http.StatusBadRequest, "board_id must be a valid integer")
		return
	}

	if err := deleteBoard(s.db, boardID); err != nil {
		slog.Error("failed to delete board setting", "boardId", boardID, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "internal server error")
		return
	}

	slog.Info("board setting deleted", "boardId", boardID)
	c.Status(http.StatusNoContent)
}

func (s *server) handleTicketEndpoint(c *gin.Context) {
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

	bs, err := s.getBoard(ticket)
	if err != nil {
		slog.Error("error getting board for ticket", "ticketId", ticket.ID, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "error getting board for ticket")
		return
	}

	if bs == nil {
		slog.Debug("no board setting found for ticket", "ticketId", ticket.ID)
		c.JSON(http.StatusOK, gin.H{"message": "no board setting found for ticket"})
		return
	}

	if !bs.Enabled {
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

func (s *server) listUsersEndpoint(c *gin.Context) {
	users, err := getAllUsers(s.db)
	if err != nil {
		slog.Error("failed to get users from database", "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to get users")
		return
	}

	c.JSON(http.StatusOK, users)
}

func (s *server) addOrUpdateUserEndpoint(c *gin.Context) {
	u := &user{}
	if err := c.ShouldBindJSON(u); err != nil {
		slog.Error("failed to bind JSON to user", "error", err)
		util.ErrorJSON(c, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedUser, err := addOrUpdateUser(s.db, u)
	if err != nil {
		slog.Error("failed to add or update user", "cwId", u.CWId, "email", u.Email, "mute", u.Mute, "ignoreUpdate", u.IgnoreUpdate, "error", err)
		util.ErrorJSON(c, http.StatusInternalServerError, "failed to add or update user")
		return
	}

	slog.Info("user added or updated", "cwId", updatedUser.CWId, "email", updatedUser.Email, "mute", updatedUser.Mute, "ignoreUpdate", updatedUser.IgnoreUpdate)
	c.JSON(http.StatusOK, updatedUser)
}

func validAction(action string) bool {
	return action == "added" || action == "updated"
}
