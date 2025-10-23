package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/webex"
)

func (c *Client) boardIDEndpoint(boardID int) string {
	return fmt.Sprintf("boards/%d", boardID)
}

func (c *Client) GetBoard(boardID int, params map[string]string) (*db.CwBoard, error) {
	return GetOne[db.CwBoard](c, c.boardIDEndpoint(boardID), params)
}

func (c *Client) ListBoards(params map[string]string) ([]db.CwBoard, error) {
	return GetMany[db.CwBoard](c, "boards", params)
}

func (c *Client) PutBoard(boardID int, board *db.CwBoard) (*db.CwBoard, error) {
	return Put[db.CwBoard](c, c.boardIDEndpoint(boardID), board)
}

func (c *Client) DeleteBoard(boardID int) error {
	return Delete(c, c.boardIDEndpoint(boardID))
}

func (c *Client) ListRooms(params map[string]string) ([]webex.Room, error) {
	return GetMany[webex.Room](c, "rooms", params)
}
