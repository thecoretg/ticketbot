package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/db"
)

func (c *Client) SyncBoards() error {
	return c.Post("sync/boards", nil, nil)
}

func (c *Client) ListBoards() ([]db.CwBoard, error) {
	return GetMany[db.CwBoard](c, "boards", nil)
}

func (c *Client) GetBoard(id int) (*db.CwBoard, error) {
	return GetOne[db.CwBoard](c, fmt.Sprintf("boards/%d", id), nil)
}
