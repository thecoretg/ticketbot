package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

func (c *Client) GetSyncStatus() (bool, error) {
	s, err := GetOne[models.SyncStatusResponse](c, "sync/status", nil)
	if err != nil {
		return false, fmt.Errorf("getting sync status: %w", err)
	}

	return s.Status, nil
}

func (c *Client) Sync(payload *models.SyncPayload) error {
	return c.Post("sync", payload, nil)
}
