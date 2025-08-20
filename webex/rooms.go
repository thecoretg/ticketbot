package webex

import "fmt"

func (c *Client) ListRooms(params map[string]string) ([]Room, error) {
	resp, err := GetOne[ListRoomsResp](c, "rooms", params)
	if err != nil {
		return nil, fmt.Errorf("listing rooms: %w", err)
	}

	return resp.Items, nil
}
