package store

import (
	"tctg-automation/internal/ticketbot/types"
)

type Store interface {
	UpsertTicket(ticket *types.Ticket) error
	GetTicket(ticketID int) (*types.Ticket, error)
	ListTickets() ([]types.Ticket, error)
}
