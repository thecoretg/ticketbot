package store

import (
	"fmt"
	"net/http"
	"tctg-automation/internal/ticketbot/types"
)

type InMemoryStore struct {
	store map[int]*types.Ticket
}

func NewInMemoryStore() *InMemoryStore {
	s := make(map[int]*types.Ticket)
	return &InMemoryStore{
		store: s,
	}
}

func (m *InMemoryStore) UpsertTicket(ticket *types.Ticket) error {
	m.store[ticket.ID] = ticket
	return nil
}

func (m *InMemoryStore) GetTicket(ticketID int) (*types.Ticket, error) {
	if ticket, exists := m.store[ticketID]; exists {
		return ticket, nil
	}
	return nil, &ErrStore{
		StatusCode: http.StatusNotFound,
		Err:        fmt.Sprintf("ticket %d not found in store", ticketID),
	}
}

func (m *InMemoryStore) ListTickets() ([]types.Ticket, error) {
	var tickets []types.Ticket
	for _, ticket := range m.store {
		tickets = append(tickets, *ticket)
	}
	return tickets, nil
}
