package sdk

import (
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) GetCurrentUser() (*models.APIUser, error) {
	return GetOne[models.APIUser](c, "users/me", nil)
}

func (c *Client) ListUsers() ([]models.APIUser, error) {
	return GetMany[models.APIUser](c, "users", nil)
}

func (c *Client) GetUser(email string) (*models.APIUser, error) {
	return GetOne[models.APIUser](c, fmt.Sprintf("users/%s", email), nil)
}

func (c *Client) CreateUser(email string) (*models.APIUser, error) {
	p := &models.APIUser{
		EmailAddress: email,
	}

	u := &models.APIUser{}

	if err := c.Post("users", p, u); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}

	return u, nil
}

func (c *Client) DeleteUser(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("users/%d", id))
}

func (c *Client) ListAPIKeys() ([]models.APIKey, error) {
	return GetMany[models.APIKey](c, "users/keys", nil)
}

func (c *Client) CreateAPIKey(email string) (string, error) {
	p := &models.CreateAPIKeyPayload{
		Email: email,
	}

	k := &models.CreateAPIKeyResponse{}
	if err := c.Post("users/keys", p, k); err != nil {
		return "", fmt.Errorf("posting to server: %w", err)
	}

	return k.Key, nil
}

func (c *Client) DeleteAPIKey(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("users/keys/%d", id))
}
