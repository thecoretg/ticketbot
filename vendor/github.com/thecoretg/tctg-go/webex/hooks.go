package webex

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) CreateWebhook(ctx context.Context, webhook *Webhook) (*Webhook, error) {
	return post[Webhook](ctx, c, "webhooks", webhook)
}

func (c *Client) GetWebhooks(ctx context.Context, params map[string]string) ([]Webhook, error) {
	resp, err := get[ListWebhooksResp](ctx, c, "webhooks", params)
	if err != nil {
		return nil, fmt.Errorf("getting list of webhooks: %w", err)
	}

	return append([]Webhook{}, resp.Items...), nil
}

func (c *Client) GetWebhook(ctx context.Context, webhookID string, params map[string]string) (*Webhook, error) {
	return get[Webhook](ctx, c, fmt.Sprintf("webhooks/%s", webhookID), params)
}

func (c *Client) PutWebhook(ctx context.Context, webhookID string, webhook *Webhook) (*Webhook, error) {
	return put[Webhook](ctx, c, fmt.Sprintf("webhooks/%s", webhookID), webhook)
}

func (c *Client) DeleteWebhook(ctx context.Context, webhookID string) error {
	return del(ctx, c, fmt.Sprintf("webhooks/%s", webhookID))
}

// ValidateWebhook checks the X-Webex-Signature header against the HMAC-SHA256 of the body.
func ValidateWebhook(r *http.Request, secret string) (bool, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return false, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("error closing request body: %v", err)
		}
	}(r.Body)

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := mac.Sum(nil)
	expectedSig := hex.EncodeToString(expectedMAC)
	actualSig := r.Header.Get("X-Webex-Signature")
	actualSig, _ = splitHeaderVals(actualSig)

	return hmac.Equal([]byte(expectedSig), []byte(actualSig)), nil
}

// splitHeaderVals returns the SHA256 and SHA512 values from the X-Webex-Signature header
func splitHeaderVals(s string) (string, string) {
	var sh2, sh5 string
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if after, ok := strings.CutPrefix(part, "SHA-256="); ok {
			sh2 = after
		} else if after, ok := strings.CutPrefix(part, "SHA-512="); ok {
			sh5 = after
		}
	}

	return sh2, sh5
}
