package store

import (
	"sync"
	"tctg-automation/internal/ticketbot/types"
)

type InMemoryStore struct {
	tickets map[int]*types.Ticket
	boards  map[int]*types.Board
	mu      sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	t := make(map[int]*types.Ticket)
	b := make(map[int]*types.Board)
	return &InMemoryStore{
		tickets: t,
		boards:  b,
	}
}

func (m *InMemoryStore) UpsertTicket(ticket *types.Ticket) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickets[ticket.ID] = ticket
	return nil
}

func (m *InMemoryStore) GetTicket(ticketID int) (*types.Ticket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if ticket, exists := m.tickets[ticketID]; exists {
		return ticket, nil
	}
	return nil, nil
}

func (m *InMemoryStore) ListTickets() ([]types.Ticket, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var tickets []types.Ticket
	for _, ticket := range m.tickets {
		tickets = append(tickets, *ticket)
	}

	if tickets == nil {
		tickets = []types.Ticket{}
	}

	return tickets, nil
}

func (m *InMemoryStore) UpsertBoard(board *types.Board) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.boards[board.ID] = board
	return nil
}

func (m *InMemoryStore) GetBoard(boardID int) (*types.Board, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if board, exists := m.boards[boardID]; exists {
		return board, nil
	}
	return nil, nil
}

func (m *InMemoryStore) ListBoards() ([]types.Board, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var boards []types.Board
	for _, board := range m.boards {
		boards = append(boards, *board)
	}

	if boards == nil {
		boards = []types.Board{}
	}

	return boards, nil
}
