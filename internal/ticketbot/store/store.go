package store

import (
	"tctg-automation/internal/ticketbot/types"
)

type Store interface {
	UpsertTicket(ticket *types.Ticket) error
	GetTicket(ticketID int) (*types.Ticket, error)
	ListTickets() ([]types.Ticket, error)
	UpsertBoard(board *types.Board) error
	GetBoard(boardID int) (*types.Board, error)
	ListBoards() ([]types.Board, error)
}
