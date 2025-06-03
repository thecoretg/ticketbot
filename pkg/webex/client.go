package webex

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"io"
	"net/http"
	"tctg-automation/pkg/aws"
)

const (
	baseUrl = "https://webexapis.com/v1"
)

type Client struct {
	httpClient *http.Client
	apiKey     string
}

func NewClient(httpClient *http.Client, apiKey string) *Client {
	return &Client{
		httpClient: httpClient,
		apiKey:     apiKey,
	}
}

func NewClientFromAWS(ctx context.Context, httpClient *http.Client, s *ssm.Client, paramName string, withDecryption bool) (*Client, error) {
	key, err := GetKeyFromAWS(ctx, s, paramName, withDecryption)
	if err != nil {
		return nil, fmt.Errorf("getting key from AWS: %w", err)
	}

	return &Client{
		httpClient: httpClient,
		apiKey:     key,
	}, nil
}

func GetKeyFromAWS(ctx context.Context, s *ssm.Client, paramName string, withDecryption bool) (string, error) {
	p, err := aws.GetParam(ctx, s, paramName, withDecryption)
	if err != nil {
		return "", err
	}

	if p.Parameter.Value == nil {
		return "", fmt.Errorf("key is nil: %w", err)
	}

	return *p.Parameter.Value, nil
}

func (c *Client) request(ctx context.Context, method, endpoint string, payload io.Reader, target interface{}) error {
	url := fmt.Sprintf("%s/%s", baseUrl, endpoint)
	req, err := http.NewRequestWithContext(ctx, method, url, payload)
	if err != nil {
		return fmt.Errorf("creating the request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	res, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending the request: %w", err)
	}

	defer res.Body.Close()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		data, err := io.ReadAll(res.Body)
		if err != nil {
			return fmt.Errorf("reading error response body: %w", err)
		}
		return fmt.Errorf("non-success response code: %d: %s", res.StatusCode, string(data))
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("reading the response body: %w", err)
	}

	if target != nil {
		if err := json.Unmarshal(data, target); err != nil {
			return fmt.Errorf("unmarshaling the response to json: %w", err)
		}
	}

	return nil
}
