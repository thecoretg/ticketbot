package sdk

import (
	"errors"
	"fmt"
	"net/http"
)

var (
	ErrNotFound = errors.New("404 status returned")
)

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
		return nil, fmt.Errorf("error response from Webex API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var allItems []T

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

	return allItems, nil
}

func Post[T any](c *Client, endpoint string, body any) (*T, error) {
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

func Put[T any](c *Client, endpoint string, body any) (*T, error) {
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
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
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
		return fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return nil
}
