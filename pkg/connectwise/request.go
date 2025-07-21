package connectwise

import (
	"fmt"
)

const (
	baseUrl = "https://api-na.myconnectwise.net/v4_6_release/apis/3.0"
)

func GetOne[T any](c *Client, endpoint string, params map[string]string) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetQueryParams(params).
		SetResult(&target).
		Get(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}
	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var target []T
	_, err := c.restClient.R().
		SetQueryParams(params).
		SetResult(&target).
		Get(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}

	return target, nil
}

func Post[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Post(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}

	return res.Result().(*T), nil
}

func Put[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Put(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}

	return res.Result().(*T), nil
}

func Patch[T any](c *Client, endpoint string, patchOps []PatchOp) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(patchOps).
		SetResult(target).
		Patch(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}

	return res.Result().(*T), nil
}

func Delete(c *Client, endpoint string) error {
	_, err := c.restClient.R().
		Delete(endpoint)

	if err != nil {
		return nil
	}

	return nil
}

func fullURL(base, endpoint string) string {
	return fmt.Sprintf("%s/%s", base, endpoint)
}
