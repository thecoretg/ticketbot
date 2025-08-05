package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"tctg-automation/pkg/connectwise"
	"time"
)

const maxConcurrentPreload = 10

func (s *server) preloadFromConnectwise(ctx context.Context, preloadBoards, preloadTickets bool) error {
	if preloadBoards {
		slog.Debug("preload boards enabled")
		time.Sleep(2 * time.Second)
		if err := s.preloadBoards(); err != nil {
			return fmt.Errorf("preloading active boards: %w", err)
		}
	}

	if preloadTickets {
		slog.Debug("preload tickets enabled")
		time.Sleep(2 * time.Second)
		if err := s.preloadOpenTickets(ctx); err != nil {
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
				b := &Board{
					ID:            board.ID,
					Name:          board.Name,
					NotifyEnabled: false,
					//WebexRooms:    []WebexRoom{},
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

func (s *server) preloadOpenTickets(ctx context.Context) error {
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
	errCh := make(chan error, len(openTickets))

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket connectwise.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.addOrUpdateTicket(ctx, ticket.ID, "preload", false); err != nil {
				errCh <- fmt.Errorf("error preloading ticket %d: %w", ticket.ID, err)
			} else {
				errCh <- nil
			}
		}(ticket)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("preloading ticket", "error", err)
			return err
		}
	}
	return nil
}
