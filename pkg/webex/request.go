package webex

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	baseURL = "https://webexapis.com/v1"
)

var ErrNotFound = errors.New("404 status received")

func GetOne[T any](c *Client, endpoint string, params map[string]string) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetQueryParams(params).
		SetResult(&target).
		Get(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsError() {
		if res.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from Webex API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var allItems []T

	endpoint = fullURL(baseURL, endpoint)
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
			return nil, fmt.Errorf("error response from Webex API: %s", res.String())
		}

		allItems = append(allItems, target...)

		params = nil
		endpoint = parseLinkHeader(res.Header().Get("Link"), "next")
	}

	return allItems, nil
}

func Put[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Put(fullURL(baseURL, endpoint))
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

func Post[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Post(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, fmt.Errorf("error response from Webex API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func Delete(c *Client, endpoint string) error {
	res, err := c.restClient.R().
		Delete(fullURL(baseURL, endpoint))
	if err != nil {
		return err
	}

	if res.IsError() {
		if res.StatusCode() == http.StatusNotFound {
			return ErrNotFound
		}
		return fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return nil
}

func fullURL(base, endpoint string) string {
	return fmt.Sprintf("%s/%s", base, endpoint)
}

func parseLinkHeader(linkHeader, rel string) string {
	for link := range strings.SplitSeq(linkHeader, ",") {
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
