package webex

import (
	"net/http"

	"resty.dev/v3"
)

type Client struct {
	restClient *resty.Client
	httpClient *http.Client
	apiKey     string
}

func NewClient(token string) *Client {
	c := resty.New()
	c.SetAuthToken(token)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)

	return &Client{restClient: c}
}
