package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

type dbNotifierResponse struct {
	ID            int          `json:"id"`
	NotifyEnabled bool         `json:"notify_enabled"`
	CreatedOn     time.Time    `json:"created_on"`
	CWBoard       db.CwBoard   `json:"cw_board"`
	WebexRoom     db.WebexRoom `json:"webex_room"`
}

func (cl *Client) handleListNotifiers(c *gin.Context) {
	notis, err := cl.Queries.ListNotifierConnections(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	if notis == nil {
		c.JSON(http.StatusOK, []dbNotifierResponse{})
		return
	}

	var resp []dbNotifierResponse
	for _, n := range notis {
		r := newDbNotifierResponse(n.NotifierConnection, n.CwBoard, n.WebexRoom)
		resp = append(resp, r)
	}

	c.JSON(http.StatusOK, resp)
}

func (cl *Client) handlePostNotifier(c *gin.Context) {
	r := &db.InsertNotifierConnectionParams{}
	if err := c.ShouldBindJSON(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json request"})
		return
	}

	if err := validateNotifierRequest(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	if _, err := cl.ensureBoardInStore(c.Request.Context(), cl.Queries, r.CwBoardID); err != nil {
		c.Error(fmt.Errorf("ensuring board %d in store: %w", r.CwBoardID, err))
		return
	}

	n, err := cl.Queries.InsertNotifierConnection(c.Request.Context(), *r)
	if err != nil {
		c.Error(err)
		return
	}

	ni, err := cl.Queries.GetNotifierConnection(c.Request.Context(), n.ID)
	if err != nil {
		c.JSON(http.StatusOK, fmt.Errorf("getting connection info (post was successful): %w", err))
		return
	}

	c.JSON(http.StatusOK, ni)
}

func (cl *Client) handleGetNotifier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("notifier_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("notifier id must be a valid integer"))
		return
	}

	n, err := cl.Queries.GetNotifierConnection(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, errorOutput(fmt.Sprintf("notifier %d not found", id)))
			return
		}
		c.Error(err)
		return
	}

	r := newDbNotifierResponse(n.NotifierConnection, n.CwBoard, n.WebexRoom)
	c.JSON(http.StatusOK, r)
}

func (cl *Client) handleDeleteNotifier(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("notifier_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorOutput("notifier id must be a valid integer"))
		return
	}

	if err := cl.Queries.DeleteNotifierConnection(c.Request.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": fmt.Sprintf("notifier with id %d does not exist", id)})
			return
		}
	}

	c.Status(http.StatusOK)
}

func validateNotifierRequest(r *db.InsertNotifierConnectionParams) error {
	if r == nil {
		return errors.New("request is nil")
	}

	if r.WebexRoomID == 0 {
		return errors.New("webex room id cannot be zero")
	}

	if r.CwBoardID == 0 {
		return errors.New("connectwise board id cannot be zero")
	}

	return nil
}

func newDbNotifierResponse(nc db.NotifierConnection, board db.CwBoard, room db.WebexRoom) dbNotifierResponse {
	return dbNotifierResponse{
		ID:            nc.ID,
		NotifyEnabled: nc.NotifyEnabled,
		CreatedOn:     nc.CreatedOn,
		CWBoard:       board,
		WebexRoom:     room}
}
