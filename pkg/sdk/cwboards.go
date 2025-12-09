package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListBoards() ([]models.Board, error) {
	return GetMany[models.Board](c, "cw/boards", nil)
}

func (c *Client) GetBoard(id int) (*models.Board, error) {
	return GetOne[models.Board](c, fmt.Sprintf("cw/boards/%d", id), nil)
}
