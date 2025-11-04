package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) handleListWebexRooms(c *gin.Context) {
	// TODO: query params?
	rooms, err := cl.Queries.ListWebexRooms(c.Request.Context())
	if err != nil {
		c.Error(fmt.Errorf("listing rooms: %w", err))
		return
	}

	if rooms == nil {
		rooms = []db.WebexRoom{}
	}

	c.JSON(http.StatusOK, rooms)
}
