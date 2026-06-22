package webex

import (
	"context"
	"fmt"
)

func (c *Client) ListPeople(ctx context.Context, email string) ([]Person, error) {
	params := map[string]string{
		"email": email,
	}

	resp, err := get[ListPeopleResp](ctx, c, "people", params)
	if err != nil {
		return nil, fmt.Errorf("listing people: %w", err)
	}

	return resp.Items, nil
}
