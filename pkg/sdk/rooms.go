package sdk

import (
	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListRecipients() ([]models.WebexRecipient, error) {
	return GetMany[models.WebexRecipient](c, "webex/rooms", nil)
}
