package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (cl *Client) handleBaseAdminPage(c *gin.Context) {
	boards, err := cl.Queries.ListBoards(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err})
		return
	}

	c.HTML(http.StatusOK, "base.gohtml", gin.H{"Title": "TicketBot Admin Dashboard", "Boards": boards})
}
