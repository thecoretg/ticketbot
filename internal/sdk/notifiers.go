package sdk

import (
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/oldserver"
)

func (c *Client) ListNotifiers() ([]oldserver.Notifier, error) {
	return GetMany[oldserver.Notifier](c, "notifiers", nil)
}

func (c *Client) GetNotifier(id int) (*oldserver.Notifier, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}

	return GetOne[oldserver.Notifier](c, fmt.Sprintf("notifiers/%d", id), nil)
}

func (c *Client) CreateNotifier(boardID, roomID int, notifyEnabled bool) (*oldserver.Notifier, error) {
	if boardID == 0 {
		return nil, errors.New("board id not provided")
	}

	if roomID == 0 {
		return nil, errors.New("room id not provided")
	}

	p := db.InsertNotifierConnectionParams{
		CwBoardID:     boardID,
		WebexRoomID:   roomID,
		NotifyEnabled: notifyEnabled,
	}

	n := &oldserver.Notifier{}
	if err := c.Post("notifiers", p, n); err != nil {
		return nil, fmt.Errorf("posting to server: %w", err)
	}

	return n, nil
}

func (c *Client) DeleteNotifier(id int) error {
	if id == 0 {
		return errors.New("no id provided")
	}

	return c.Delete(fmt.Sprintf("notifiers/%d", id))
}
