package webex

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (c *Client) CreateWebhook(webhook *Webhook) (*Webhook, error) {
	return Post[Webhook](c, "webhooks", webhook)
}

func (c *Client) GetWebhooks(params map[string]string) ([]Webhook, error) {
	resp, err := GetOne[ListWebhooksResp](c, "webhooks", params)
	if err != nil {
		return nil, fmt.Errorf("getting list of webhooks: %w", err)
	}

	w := append([]Webhook{}, resp.Items...)
	return w, nil
}

func (c *Client) GetWebhook(webhookID string, params map[string]string) (*Webhook, error) {
	return GetOne[Webhook](c, fmt.Sprintf("webhooks/%s", webhookID), params)
}

func (c *Client) PutWebhook(webhookID string, webhook *Webhook) (*Webhook, error) {
	return Put[Webhook](c, fmt.Sprintf("webhooks/%s", webhookID), webhook)
}

func (c *Client) DeleteWebhook(webhookID string) error {
	return Delete(c, fmt.Sprintf("webhooks/%s", webhookID))
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
	// shortened values as to not collide with packages
	var sh2, sh5 string
	for part := range strings.SplitSeq(s, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "SHA-256=") {
			sh2 = strings.TrimPrefix(part, "SHA-256=")
		} else if strings.HasPrefix(part, "SHA-512=") {
			sh5 = strings.TrimPrefix(part, "SHA-512=")
		}
	}

	return sh2, sh5
}
