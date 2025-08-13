package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
	"log/slog"
	"sync"
	"time"
)

func (s *Server) PreloadData(ctx context.Context, preloadBoards, preloadTickets bool, maxConcurrent int) error {
	if preloadBoards {
		slog.Debug("preload boards enabled")
		time.Sleep(2 * time.Second)
		if err := s.preloadBoards(ctx, maxConcurrent); err != nil {
			return fmt.Errorf("preloading active boards: %w", err)
		}
	}

	if preloadTickets {
		slog.Debug("preload tickets enabled")
		time.Sleep(2 * time.Second)
		if err := s.preloadOpenTickets(ctx, maxConcurrent); err != nil {
			return fmt.Errorf("preloading open tickets: %w", err)
		}
	}

	return nil
}

func (s *Server) preloadBoards(ctx context.Context, maxConcurrent int) error {
	params := map[string]string{
		"conditions": "inactiveFlag = false",
	}

	slog.Info("loading existing boards")
	boards, err := s.CWClient.ListBoards(params)
	if err != nil {
		return fmt.Errorf("getting boards from CW: %w", err)
	}
	slog.Info("got boards", "total_boards", len(boards))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for _, board := range boards {
		_, err := s.Queries.GetBoard(ctx, board.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Info("board not found in data store - adding", "board_id", board.ID, "board_name", board.Name)
				sem <- struct{}{}
				wg.Add(1)
				go func(board connectwise.Board) {
					defer wg.Done()
					defer func() { <-sem }()
					p := db.InsertBoardParams{
						ID:            board.ID,
						Name:          board.Name,
						NotifyEnabled: false,
						WebexRoomID:   nil,
					}
					if _, err := s.Queries.InsertBoard(ctx, p); err != nil {
						slog.Warn("error preloading board", "board_id", board.ID, "error", err)
					}
					slog.Info("preloaded board", "board_id", board.ID, "board_name", board.Name)
				}(board)
			} else {
				slog.Warn("an error occured trying to check if a board exists", "board_id", board.ID, "board_name", board.Name, "error", err)
			}
		} else {
			slog.Info("board is already in data store", "board_id", board.ID, "board_name", board.Name)
		}
	}

	wg.Wait()
	return nil
}

func (s *Server) preloadOpenTickets(ctx context.Context, maxConcurrent int) error {
	params := map[string]string{
		"pageSize":   "100",
		"conditions": "closedFlag = false",
	}

	slog.Info("loading existing open tickets")
	openTickets, err := s.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from CW: %w", err)
	}
	slog.Info("got open tickets", "total_tickets", len(openTickets))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	errCh := make(chan error, len(openTickets))

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket connectwise.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := s.processTicketPayload(ctx, ticket.ID, "preload", true, false); err != nil {
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
			if s.Config.ExitOnError {
				slog.Info("exiting because EXIT_ON_ERROR is enabled")
				return err
			}
		}
	}
	return nil
}
