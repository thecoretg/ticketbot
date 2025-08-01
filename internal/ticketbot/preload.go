package ticketbot

import (
	"fmt"
	"log/slog"
	"sync"
	"tctg-automation/internal/ticketbot/types"
	"tctg-automation/pkg/connectwise"
)

const maxConcurrentPreload = 10

func (s *server) preloadFromConnectwise(preloadBoards, preloadTickets bool) error {
	if preloadBoards {
		if err := s.preloadBoards(); err != nil {
			return fmt.Errorf("preloading active boards: %w", err)
		}
	}

	if preloadTickets {
		if err := s.preloadOpenTickets(); err != nil {
			return fmt.Errorf("preloading open tickets: %w", err)
		}
	}

	return nil
}

func (s *server) preloadBoards() error {
	params := map[string]string{
		"conditions": "inactiveFlag = false",
	}

	slog.Info("loading existing boards")
	boards, err := s.cwClient.ListBoards(params)
	if err != nil {
		return fmt.Errorf("getting boards from CW: %w", err)
	}
	slog.Info("got boards", "total_boards", len(boards))
	sem := make(chan struct{}, maxConcurrentPreload)
	var wg sync.WaitGroup
	for _, board := range boards {
		storeBoard, _ := s.dataStore.GetBoard(board.ID)
		if storeBoard == nil {
			slog.Info("board not found in data store - adding", "board_id", board.ID, "board_name", board.Name)
			sem <- struct{}{}
			wg.Add(1)
			go func(board connectwise.Board) {
				defer wg.Done()
				defer func() { <-sem }()
				b := &types.Board{
					ID:            board.ID,
					Name:          board.Name,
					NotifyEnabled: false,
					WebexRoomIDs:  nil,
				}
				if err := s.dataStore.UpsertBoard(b); err != nil {
					slog.Warn("error preloading board", "board_id", board.ID, "error", err)
				}
				slog.Info("preloaded board", "board_id", board.ID, "board_name", board.Name)
			}(board)
		} else {
			slog.Info("board is already in data store", "board_id", board.ID, "board_name", board.Name)
		}
	}

	wg.Wait()
	return nil
}

func (s *server) preloadOpenTickets() error {
	params := map[string]string{
		"pageSize":   "100",
		"conditions": "closedFlag = false and board/id = 34",
	}

	slog.Info("loading existing open tickets")
	openTickets, err := s.cwClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from CW: %w", err)
	}
	slog.Info("got open tickets", "total_tickets", len(openTickets))
	sem := make(chan struct{}, maxConcurrentPreload)
	var wg sync.WaitGroup

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket connectwise.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			storeTicket, _ := s.dataStore.GetTicket(ticket.ID)
			if err := s.addOrUpdateTicket(storeTicket, &ticket, false); err != nil {
				slog.Warn("error preloading open ticket", "ticket_id", ticket.ID, "error", err)
			}
		}(ticket)
	}

	wg.Wait()
	return nil
}
