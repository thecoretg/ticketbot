package webex

import (
	"context"
	"fmt"
)

func (c *Client) ListRooms(ctx context.Context, params map[string]string) ([]Room, error) {
	resp, err := get[ListRoomsResp](ctx, c, "rooms", params)
	if err != nil {
		return nil, fmt.Errorf("listing rooms: %w", err)
	}

	return resp.Items, nil
}
