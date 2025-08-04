package connectwise

import (
	"fmt"
	"strings"
)

// TODO: implement retry handling
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

	if res.IsError() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func GetMany[T any](c *Client, endpoint string, params map[string]string) ([]T, error) {
	var allItems []T

	endpoint = fullURL(baseUrl, endpoint)
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

func Post[T any](c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetBody(body).
		SetResult(target).
		Post(fullURL(baseUrl, endpoint))

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
		Put(fullURL(baseUrl, endpoint))

	if err != nil {
		return nil, err
	}

	if res.IsError() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
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

	if res.IsError() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return res.Result().(*T), nil
}

func Delete(c *Client, endpoint string) error {
	res, err := c.restClient.R().
		Delete(fullURL(baseUrl, endpoint))

	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return nil
}

func fullURL(base, endpoint string) string {
	return fmt.Sprintf("%s/%s", base, endpoint)
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
