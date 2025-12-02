package sdk

import (
	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListRooms() ([]models.WebexRecipient, error) {
	return GetMany[models.WebexRecipient](c, "webex/rooms", nil)
}
