package webex

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/thecoretg/ticketbot/amazon"
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
	p, err := amazon.GetParam(ctx, s, paramName, withDecryption)
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

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Printf("error closing response body: %v", err)
		}
	}(res.Body)
	if res.StatusCode < 200 || res.StatusCode >= 300 {
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
