package oldserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Work in progress...

func (cl *Client) handleHomePage(c *gin.Context) {
	c.HTML(http.StatusOK, "home.gohtml", gin.H{"Title": "TicketBot Admin"})
}

func (cl *Client) handleBoardsPage(c *gin.Context) {
	boards, err := cl.Queries.ListBoards(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.HTML(http.StatusOK, "boards.gohtml", gin.H{"Title": "Boards", "Boards": boards})
}

func (cl *Client) handleNotifiersPage(c *gin.Context) {
	notifiers, err := cl.Queries.ListNotifierConnections(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	c.HTML(http.StatusOK, "notifiers.gohtml", gin.H{"Title": "Notifier Connections", "Notifiers": notifiers})
}
