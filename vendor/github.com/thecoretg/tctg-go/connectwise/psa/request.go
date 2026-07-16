package psa

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	baseURL = "https://api-na.myconnectwise.net/v4_6_release/apis/3.0"
)

var ErrNotFound = errors.New("404 status returned")

func get[T any](ctx context.Context, c *Client, endpoint string, params map[string]string) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetContext(ctx).
		SetQueryParams(params).
		SetResult(&target).
		Get(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		if res.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return &target, nil
}

func getMany[T any](ctx context.Context, c *Client, endpoint string, params map[string]string) ([]T, error) {
	var allItems []T

	endpoint = fullURL(baseURL, endpoint)
	for endpoint != "" {
		var target []T
		res, err := c.restClient.R().
			SetContext(ctx).
			SetQueryParams(params).
			SetResult(&target).
			Get(endpoint)
		if err != nil {
			return nil, err
		}

		if res.IsStatusFailure() {
			if res.StatusCode() == http.StatusNotFound {
				return nil, ErrNotFound
			}
			return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
		}

		allItems = append(allItems, target...)
		params = nil
		endpoint = parseLinkHeader(res.Header().Get("Link"), "next")
	}

	return allItems, nil
}

func post[T any](ctx context.Context, c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&target).
		Post(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return &target, nil
}

func put[T any](ctx context.Context, c *Client, endpoint string, body any) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetContext(ctx).
		SetBody(body).
		SetResult(&target).
		Put(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		if res.StatusCode() == http.StatusNotFound {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return &target, nil
}

func patch[T any](ctx context.Context, c *Client, endpoint string, patchOps []PatchOp) (*T, error) {
	var target T
	res, err := c.restClient.R().
		SetContext(ctx).
		SetBody(patchOps).
		SetResult(&target).
		Patch(fullURL(baseURL, endpoint))
	if err != nil {
		return nil, err
	}

	if res.IsStatusFailure() {
		return nil, fmt.Errorf("error response from ConnectWise API: %s", res.String())
	}

	return &target, nil
}

func del(ctx context.Context, c *Client, endpoint string) error {
	res, err := c.restClient.R().
		SetContext(ctx).
		Delete(fullURL(baseURL, endpoint))
	if err != nil {
		return err
	}

	if res.IsStatusFailure() {
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
