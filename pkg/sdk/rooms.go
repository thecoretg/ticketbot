package sdk

import (
	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListRooms() ([]models.WebexRoom, error) {
	return GetMany[models.WebexRoom](c, "webex/rooms", nil)
}
