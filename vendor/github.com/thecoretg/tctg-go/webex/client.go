package webex

import (
	"context"
	"fmt"
	"os"
	"strings"

	"resty.dev/v3"
)

type Config struct {
	Token string
}

type Client struct {
	restClient *resty.Client
}

// NewClient builds a Webex client from cfg. The ctx parameter is accepted for
// signature consistency with the other tctg-go clients; Webex uses a static
// bearer token and performs no setup that requires it.
func NewClient(_ context.Context, cfg Config) (*Client, error) {
	var missing []string
	if cfg.Token == "" {
		missing = append(missing, "Token")
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("missing webex config fields: %s", strings.Join(missing, ", "))
	}

	c := resty.New()
	c.SetAuthToken(cfg.Token)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)
	c.SetLoggerWarnLevel(false)

	return &Client{restClient: c}, nil
}

func NewClientFromEnv(ctx context.Context) (*Client, error) {
	return NewClient(ctx, Config{
		Token: os.Getenv("WEBEX_TOKEN"),
	})
}
