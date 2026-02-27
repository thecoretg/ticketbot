package sdk

import (
	"github.com/thecoretg/ticketbot/models"
)

func (c *Client) ListRecipients() ([]models.WebexRecipient, error) {
	return GetMany[models.WebexRecipient](c, "webex/rooms", nil)
}
