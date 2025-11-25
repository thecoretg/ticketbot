package sdk

import (
	"fmt"

	"github.com/thecoretg/ticketbot/internal/oldserver"
)

func (c *Client) GetConfig() (*oldserver.AppConfig, error) {
	return GetOne[oldserver.AppConfig](c, "config", nil)
}

func (c *Client) UpdateConfig(params *oldserver.AppConfigPayload) (*oldserver.AppConfig, error) {
	cfg := &oldserver.AppConfig{}
	if err := c.Put("config", params, cfg); err != nil {
		return nil, fmt.Errorf("sending update request: %w", err)
	}

	return cfg, nil
}

func (c *Client) GetAppState() (*oldserver.AppState, error) {
	return GetOne[oldserver.AppState](c, "state", nil)
}
