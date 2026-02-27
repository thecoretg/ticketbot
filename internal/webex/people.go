package webex

import "fmt"

func (c *Client) ListPeople(email string) ([]Person, error) {
	params := map[string]string{
		"email": email,
	}

	resp, err := GetOne[ListPeopleResp](c, "people", params)
	if err != nil {
		return nil, fmt.Errorf("listing people: %w", err)
	}

	return resp.Items, nil
}
