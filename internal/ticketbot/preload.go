package ticketbot

import (
	"fmt"
	"log/slog"
	"sync"
	"tctg-automation/pkg/connectwise"
)

const maxConcurrentPreload = 10

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
			if err := s.addOrUpdateTicket(nil, &ticket, false); err != nil {
				slog.Warn("error preloading open ticket", "ticket_id", ticket.ID, "error", err)
			}
		}(ticket)
	}

	wg.Wait()
	return nil
}
