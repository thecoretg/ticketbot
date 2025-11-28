package sdk

import (
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListNotifierRules() ([]models.NotifierRule, error) {
	return GetMany[models.NotifierRule](c, "notifiers/rules", nil)
}

func (c *Client) GetNotifierRule(id int) (*models.NotifierRule, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}

	return GetOne[models.NotifierRule](c, fmt.Sprintf("notifiers/rules/%d", id), nil)
}

func (c *Client) CreateNotifierRule(payload *models.NotifierRule) (*models.NotifierRule, error) {
	n := &models.NotifierRule{}
	if err := c.Post("notifiers/rules", payload, n); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}

	return n, nil
}

func (c *Client) DeleteNotifierRule(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("notifiers/rules/%d", id))
}

func (c *Client) ListUserForwards() ([]models.UserForward, error) {
	return GetMany[models.UserForward](c, "notifiers/forwards", nil)
}

func (c *Client) GetUserForward(id int) (*models.UserForward, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}

	return GetOne[models.UserForward](c, fmt.Sprintf("notifiers/forwards/%d", id), nil)
}

func (c *Client) CreateUserForward(payload *models.UserForward) (*models.UserForward, error) {
	uf := &models.UserForward{}
	if err := c.Post("notifiers/forwards", payload, uf); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}

	return uf, nil
}

func (c *Client) DeleteUserForward(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("notifiers/forwards/%d", id))
}
