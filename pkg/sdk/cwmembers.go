package sdk

import (
	"github.com/thecoretg/ticketbot/internal/models"
)

func (c *Client) ListMembers() ([]models.Member, error) {
	return GetMany[models.Member](c, "cw/members", nil)
}
