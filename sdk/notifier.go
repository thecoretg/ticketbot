package sdk

import (
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

func (c *Client) ListUserForwards(params map[string]string) ([]models.NotifierForwardFull, error) {
	return GetMany[models.NotifierForwardFull](c, "notifiers/forwards", params)
}

func (c *Client) GetUserForward(id int) (*models.NotifierForward, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}

	return GetOne[models.NotifierForward](c, fmt.Sprintf("notifiers/forwards/%d", id), nil)
}

func (c *Client) CreateUserForward(payload *models.NotifierForward) (*models.NotifierForward, error) {
	uf := &models.NotifierForward{}
	if err := c.Post("notifiers/forwards", payload, uf); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}

	return uf, nil
}

func (c *Client) UpdateUserForward(id int, payload *models.NotifierForward) (*models.NotifierForward, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}

	uf := &models.NotifierForward{}
	if err := c.Put(fmt.Sprintf("notifiers/forwards/%d", id), payload, uf); err != nil {
		return nil, fmt.Errorf("sending update request: %w", err)
	}

	return uf, nil
}

func (c *Client) DeleteUserForward(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("notifiers/forwards/%d", id))
}
