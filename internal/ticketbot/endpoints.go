package ticketbot

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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
