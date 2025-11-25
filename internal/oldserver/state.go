package oldserver

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type AppState struct {
	SyncingTickets    bool `json:"syncing_tickets"`
	SyncingWebexRooms bool `json:"syncing_webex_rooms"`
	SyncingBoards     bool `json:"syncing_boards"`
}

var defaultAppState = &AppState{
	SyncingTickets:    false,
	SyncingWebexRooms: false,
	SyncingBoards:     false,
}

func (cl *Client) handleGetState(c *gin.Context) {
	as, err := cl.getAppState()
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, as)
}

func (cl *Client) getAppState() (*AppState, error) {
	cl.setStateIfNil()
	return cl.State, nil
}

func (cl *Client) setSyncingTickets(syncing bool) {
	cl.setStateIfNil()
	cl.State.SyncingTickets = syncing
}

func (cl *Client) setSyncingWebexRooms(syncing bool) {
	cl.setStateIfNil()
	cl.State.SyncingWebexRooms = syncing
}

func (cl *Client) setSyncingBoards(syncing bool) {
	cl.setStateIfNil()
	cl.State.SyncingBoards = syncing
}

func (cl *Client) setStateIfNil() {
	if cl.State == nil {
		cl.State = defaultAppState
	}
}
