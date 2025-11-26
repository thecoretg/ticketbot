package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) GetConfig() (*models.Config, error) {
	return GetOne[models.Config](c, "config", nil)
}

func (c *Client) UpdateConfig(params *models.Config) (*models.Config, error) {
	cfg := &models.Config{}
	if err := c.Put("config", params, cfg); err != nil {
		return nil, fmt.Errorf("sending update request: %w", err)
	}

	return cfg, nil
}
