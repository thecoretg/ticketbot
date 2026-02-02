package webex

import (
	"resty.dev/v3"
)

type Client struct {
	restClient *resty.Client
}

func NewClient(token string) *Client {
	c := resty.New()
	c.SetAuthToken(token)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)
	c.SetDisableWarn(true)

	return &Client{restClient: c}
}
