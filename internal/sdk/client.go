package sdk

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"resty.dev/v3"
)

type Client struct {
	restClient *resty.Client
}

func NewClient(apiKey, baseURL string) (*Client, error) {
	if apiKey == "" {
		return nil, errors.New("api key is blank")
	}
	c := resty.New()
	c.SetBaseURL(baseURL)
	c.SetAuthToken(apiKey)
	c.SetHeader("Content-Type", "application/json")
	c.SetHeader("Accept", "application/json")
	c.SetRetryCount(3)

	return &Client{restClient: c}, nil
}

var (
	ErrNotFound = errors.New("404 status returned")
)

func (c *Client) TestConnection() error {
	res, err := c.restClient.R().
		Get("ping")

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from ticketbot api: %s", res.String())
	}

	return nil
}

func GetOne[T any](c *Client, endpoint string, params map[string]string) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetQueryParams(params).
		SetResult(&target).
		Get(endpoint)

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		if res.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var allItems []T

	for endpoint != "" {
		var target []T
		req := c.restClient.R().
			SetQueryParams(params).
			SetResult(&target)

		res, err := req.Get(endpoint)
		if err != nil {
			return nil, err
		}

		if res.IsError() {
			if res.StatusCode() == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
		}

		for _, item := range target {
			allItems = append(allItems, item)
		}

		params = nil
		endpoint = parseLinkHeader(res.Header().Get("Link"), "next")
	}

	return allItems, nil
}

func Post(c *Client, endpoint string, body any) error {
	res, err := c.restClient.R().
		SetBody(body).
		Post(endpoint)

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from API: %s", res.String())
	}

	return nil
}

func PostWithReturn[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Post(endpoint)

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func Put(c *Client, endpoint string, body any) error {
	res, err := c.restClient.R().
		SetBody(body).
		Post(endpoint)

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from API: %s", res.String())
	}

	return nil
}

func PutWithReturn[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Put(endpoint)

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		if res.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func Delete(c *Client, endpoint string) error {
	res, err := c.restClient.R().
		Delete(endpoint)

	if err != nil {
		return err
	}

	if res.IsError() {
		if res.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("error response from API: %s", res.String())
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
