package sdk

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/thecoretg/ticketbot/models"
)

// TicketQuery mirrors the GET /tickets filters. Zero values are omitted.
type TicketQuery struct {
	BoardID        int
	StatusID       int
	Closed         *bool
	IncludeDeleted bool
	Search         string
	Page           int
	PageSize       int
}

func (q TicketQuery) params() map[string]string {
	p := map[string]string{}
	if q.BoardID != 0 {
		p["board_id"] = strconv.Itoa(q.BoardID)
	}
	if q.StatusID != 0 {
		p["status_id"] = strconv.Itoa(q.StatusID)
	}
	if q.Closed != nil {
		p["closed"] = strconv.FormatBool(*q.Closed)
	}
	if q.IncludeDeleted {
		p["deleted"] = "true"
	}
	if q.Search != "" {
		p["q"] = q.Search
	}
	if q.Page > 0 {
		p["page"] = strconv.Itoa(q.Page)
	}
	if q.PageSize > 0 {
		p["page_size"] = strconv.Itoa(q.PageSize)
	}
	return p
}

func (c *Client) ListTickets(q TicketQuery) (*models.TicketPage, error) {
	return GetOne[models.TicketPage](c, "tickets", q.params())
}

func (c *Client) GetTicket(id int) (*models.TicketDetail, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}
	return GetOne[models.TicketDetail](c, fmt.Sprintf("tickets/%d", id), nil)
}

func (c *Client) GetTicketRaw(id int) (json.RawMessage, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}
	raw, err := GetOne[json.RawMessage](c, fmt.Sprintf("tickets/%d/raw", id), nil)
	if err != nil {
		return nil, err
	}
	return *raw, nil
}

// ListTicketEvents pages backwards through a ticket's events, newest first.
func (c *Client) ListTicketEvents(id int, limit int, beforeID int64) ([]models.TicketEvent, error) {
	if id == 0 {
		return nil, errors.New("no id provided")
	}
	p := map[string]string{}
	if limit > 0 {
		p["limit"] = strconv.Itoa(limit)
	}
	if beforeID > 0 {
		p["before_id"] = strconv.FormatInt(beforeID, 10)
	}
	return GetMany[models.TicketEvent](c, fmt.Sprintf("tickets/%d/events", id), p)
}
