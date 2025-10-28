package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
)

func (cl *Client) handlePreload(c *gin.Context) {
	c.Status(http.StatusOK)

	go func() {
		if err := cl.preloadConnectwiseData(context.Background()); err != nil {
			slog.Error("preloading data", "error", err)
		}
	}()
}

func (cl *Client) preloadConnectwiseData(ctx context.Context) error {
	if err := cl.preloadBoards(ctx, cl.Config.MaxConcurrentPreloads); err != nil {
		return fmt.Errorf("preloading active boards: %w", err)
	}

	if err := cl.preloadOpenTickets(ctx, cl.Config.MaxConcurrentPreloads); err != nil {
		return fmt.Errorf("preloading open tickets: %w", err)
	}

	return nil
}

func (cl *Client) preloadBoards(ctx context.Context, maxConcurrent int) error {
	slog.Debug("beginning preloading boards")
	params := map[string]string{
		"conditions": "inactiveFlag = false",
	}

	slog.Debug("loading existing boards from connectwise")
	boards, err := cl.CWClient.ListBoards(params)
	if err != nil {
		return fmt.Errorf("getting boards from CW: %w", err)
	}
	slog.Debug("got boards from connectwise", "total_boards", len(boards))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for _, board := range boards {
		_, err := cl.Queries.GetBoard(ctx, board.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Debug("board not found in data store - adding", "board_id", board.ID, "board_name", board.Name)
				sem <- struct{}{}
				wg.Add(1)
				go func(board psa.Board) {
					defer wg.Done()
					defer func() { <-sem }()
					p := db.InsertBoardParams{
						ID:   board.ID,
						Name: board.Name,
					}
					if _, err := cl.Queries.InsertBoard(ctx, p); err != nil {
						slog.Warn("error preloading board", "board_id", board.ID, "error", err)
					}
					slog.Debug("preloaded board", "board_id", board.ID, "board_name", board.Name)
				}(board)
			} else {
				slog.Warn("an error occured trying to check if a board exists", "board_id", board.ID, "board_name", board.Name, "error", err)
			}
		} else {
			slog.Debug("board is already in data store", "board_id", board.ID, "board_name", board.Name)
		}
	}

	wg.Wait()
	return nil
}

func (cl *Client) preloadMembers(ctx context.Context, maxConcurrent int) error {
	slog.Debug("beginning preloading members")
	slog.Debug("loading existing members from connectwise")

	members, err := cl.CWClient.ListMembers(nil)
	if err != nil {
		return fmt.Errorf("getting members from CW: %w", err)
	}
	slog.Debug("got members from connectwise", "total_members", len(members))

	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	for _, member := range members {
		_, err := cl.Queries.GetMember(ctx, member.ID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				slog.Debug("member not found in data store - adding", "member_id", member.ID, "member_identifier", member.Identifier)
				sem <- struct{}{}
				wg.Add(1)
				go func(member psa.Member) {
					defer wg.Done()
					defer func() { <-sem }()
					p := db.InsertMemberParams{
						ID:           member.ID,
						Identifier:   member.Identifier,
						FirstName:    member.FirstName,
						LastName:     member.LastName,
						PrimaryEmail: member.PrimaryEmail,
					}
					if _, err := cl.Queries.InsertMember(ctx, p); err != nil {
						slog.Warn("error preloading member", "member_id", member.ID, "member_identifier", member.Identifier, "error", err)
					}
					slog.Debug("preloaded member", "member_id", member.ID, "member_identifier", member.Identifier)
				}(member)
			} else {
				slog.Warn("an error occured trying to check if a member exists", "member_id", member.ID, "member_identifier", member.Identifier, "error", err)
			}
		} else {
			slog.Debug("member is already in data store", "member_id", member.ID, "member_identifier", member.Identifier)
		}
	}

	wg.Wait()
	return nil
}

// preloadOpenTickets finds all open tickets in Connectwise and loads them into the DB if they don't
// already exist. It does not attempt to notify since that would result in tons of notifications
// for already existing tickets.
func (cl *Client) preloadOpenTickets(ctx context.Context, maxConcurrent int) error {
	slog.Debug("beginning preloading tickets")
	params := map[string]string{
		"pageSize":   "100",
		"conditions": "closedFlag = false",
	}

	slog.Debug("loading existing open tickets")
	openTickets, err := cl.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from CW: %w", err)
	}
	slog.Debug("got open tickets from connectwise", "total_tickets", len(openTickets))
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup
	errCh := make(chan error, len(openTickets))

	for _, ticket := range openTickets {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket psa.Ticket) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := cl.processTicket(ctx, ticket.ID, "preload", true); err != nil {
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
			if cl.Config.ExitOnError {
				slog.Debug("exiting because EXIT_ON_ERROR is enabled")
				return err
			}
		}
	}
	return nil
}
