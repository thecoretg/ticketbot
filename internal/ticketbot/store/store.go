package store

import (
	"fmt"
	"tctg-automation/internal/ticketbot/types"
)

type Store interface {
	UpsertTicket(ticket *types.Ticket) error
	GetTicket(ticketID int) (*types.Ticket, error)
	ListTickets() ([]types.Ticket, error)
}

type ErrStore struct {
	StatusCode int
	Err        string
}

func (e *ErrStore) Error() string {
	return fmt.Sprintf("Status %d: %v", e.StatusCode, e.Err)
}
