package sdk

import (
	"fmt"
	"strings"

	"resty.dev/v3"
)

type Client struct {
	restClient *resty.Client
}

type APIError struct {
	Message string `json:"error"`
}

func (e *APIError) Error() string {
	return e.Message
}

func NewClient(apiKey, baseURL string) (*Client, error) {
	c := resty.New()
	c.SetBaseURL(baseURL)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)

	if apiKey != "" {
		c.SetAuthToken(apiKey)
	}

	return &Client{restClient: c}, nil
}

func (c *Client) Ping() error {
	var apiErr APIError
	res, err := c.restClient.R().
		SetError(&apiErr).
		Get("")

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from ticketbot api: %w", err)
	}

	return nil
}

func (c *Client) AuthTest() error {
	var apiErr APIError
	res, err := c.restClient.R().
		SetError(&apiErr).
		Get("authtest")

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from ticketbot api: %s", res.String())
	}

	return nil
}

func GetOne[T any](c *Client, endpoint string, params map[string]string) (*T, error) {
	var (
		target T
		apiErr APIError
	)

	res, err := c.restClient.R().
		SetQueryParams(params).
		SetResult(&target).
		SetError(&apiErr).
		Get(endpoint)

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, &apiErr
	}

	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var (
		allItems []T
		apiErr   APIError
	)

	for endpoint != "" {
		var target []T
		req := c.restClient.R().
			SetQueryParams(params).
			SetError(&apiErr).
			SetResult(&target)

		res, err := req.Get(endpoint)
		if err != nil {
			return nil, err
		}

		if res.IsError() {
			return nil, &apiErr
		}

		for _, item := range target {
			allItems = append(allItems, item)
		}

		params = nil
		endpoint = parseLinkHeader(res.Header().Get("Link"), "next")
	}

	return allItems, nil
}

func (c *Client) Post(endpoint string, body, target any) error {
	var apiErr APIError
	req := c.restClient.R().
		SetError(&apiErr).
		SetBody(body)

	if target != nil {
		req.SetResult(target)
	}

	res, err := req.Post(endpoint)
	if err != nil {
		return err
	}

	if res.IsError() {
		return &apiErr
	}

	return nil
}

func (c *Client) Put(endpoint string, body, target any) error {
	var apiErr APIError
	req := c.restClient.R().
		SetBody(body).
		SetError(&apiErr)

	if target != nil {
		req.SetResult(target)
	}

	res, err := req.Put(endpoint)
	if err != nil {
		return err
	}

	if res.IsError() {
		return &apiErr
	}

	return nil
}

func (c *Client) Delete(endpoint string) error {
	var apiErr APIError
	res, err := c.restClient.R().
		SetError(&apiErr).
		Delete(endpoint)

	if err != nil {
		return err
	}

	if res.IsError() {
		return &apiErr
	}

	return nil
}

func parseLinkHeader(linkHeader, rel string) string {
	links := strings.Split(linkHeader, ",")
	for _, link := range links {
		parts := strings.Split(strings.TrimSpace(link), ";")
		if len(parts) < 2 {
			continue
		}
		urlPart := strings.Trim(parts[0], "<>")
		relPart := strings.TrimSpace(parts[1])
		if relPart == fmt.Sprintf(`rel="%s"`, rel) {
			return urlPart
		}
	}

	return ""
}
