package sdk

import "github.com/thecoretg/ticketbot/internal/models"

func (c *Client) Sync(payload *models.SyncPayload) error {
	return c.Post("sync", payload, nil)
}
