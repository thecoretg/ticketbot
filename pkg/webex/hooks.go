package webex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
)

func (c *Client) CreateWebhook(ctx context.Context, newWebhook *Webhook) (*Webhook, error) {
	j, err := json.Marshal(newWebhook)
	if err != nil {
		return nil, fmt.Errorf("marshaling new webhook to json: %w", err)
	}

	p := bytes.NewReader(j)
	w := &Webhook{}
	if err := c.request(ctx, "POST", "webhooks", p, w); err != nil {
		return nil, fmt.Errorf("posting new webhook: %w", err)
	}

	return w, nil
}

func (c *Client) GetAllWebhooks(ctx context.Context) ([]Webhook, error) {
	w := &WebhooksGetResponse{}
	if err := c.request(ctx, "GET", "webhooks", nil, w); err != nil {
		return nil, fmt.Errorf("getting all webex webhooks: %w", err)
	}

	return w.Items, nil
}

func (c *Client) GetWebhook(ctx context.Context, webhookId string) (*Webhook, error) {
	w := &Webhook{}
	if err := c.request(ctx, "GET", fmt.Sprintf("webooks/%s", webhookId), nil, w); err != nil {
		return nil, fmt.Errorf("getting webex webhook %s: %w", webhookId, err)
	}

	return w, nil
}

func (c *Client) DeleteWebhook(ctx context.Context, webhookId string) error {
	if err := c.request(ctx, "DELETE", fmt.Sprintf("webhooks/%s", webhookId), nil, nil); err != nil {
		return fmt.Errorf("deleting webex webhook %s: %w", err)
	}

	return nil
}
