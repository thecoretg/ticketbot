package sdk

import (
	"github.com/thecoretg/ticketbot/models"
)

func (c *Client) ListMembers() ([]models.Member, error) {
	return GetMany[models.Member](c, "cw/members", nil)
}
