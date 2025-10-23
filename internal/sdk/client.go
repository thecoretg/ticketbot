package sdk

import (
	"errors"
	"fmt"

	"resty.dev/v3"
)

type Client struct {
	restClient *resty.Client
}

func NewClient(key, baseURL string) (*Client, error) {
	if key == "" {
		return nil, errors.New("api key cannot be blank")
	}

	if baseURL == "" {
		return nil, errors.New("base URL cannot be blank")
	}
	c := resty.New()
	c.SetAuthToken(key)
	c.SetBaseURL(baseURL)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)

	return &Client{restClient: c}, nil
}

func (c *Client) TestConnection() error {
	res, err := c.restClient.R().
		Get("ping")

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("testing connection: %s", res.String())
	}

	return nil
}
