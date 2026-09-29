package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/models"
)

// SyncOpenTickets reports into ph, one step per open ticket processed.
// SyncOpenTickets fetches every open ticket on boardIDs (all boards when empty). runRules runs
// each ticket's workflow when the fetch finds a change, as a webhook would.
func (s *Service) SyncOpenTickets(ctx context.Context, boardIDs []int, maxSyncs int, runRules bool, ph *phase) error {
	start := time.Now()
	slog.Info("cwsvc: beginning ticket sync", "board_ids", boardIDs)
	defer func() {
		slog.Info("sync: syncing tickets complete", "took_time", time.Since(start))
	}()

	// First phase: get all OPEN tickets from connectwise and ensure they all exist
	// in the database.
	con := "closedFlag = false"
	if len(boardIDs) > 0 {
		con += fmt.Sprintf(" AND %s", boardIDParam(boardIDs))
	}

	params := map[string]string{
		"pageSize":   "100",
		"conditions": con,
	}

	tix, err := s.CW.CWClient.ListTickets(ctx, params)
	if err != nil {
		return fmt.Errorf("getting open tickets from connectwise: %w", err)
	}
	slog.Info("cwsvc: open ticket sync: got open tickets from connectwise", "total_tickets", len(tix))
	ph.counting("Processing tickets", len(tix))
	errCh := make(chan error, len(tix))

	// A cancel stops new tickets from starting. The ones already running finish on a context
	// the cancel does not reach, so a saved ticket always gets its sync event.
	running := context.WithoutCancel(ctx)
	stopped := forEach(ctx, tix, maxSyncs, func(ticket psa.Ticket) {
		opts := ticketbot.ProcessOpts{Source: models.SourceSync, RunRules: runRules}
		if err := s.Ticketbot.ProcessTicket(running, ticket.ID, opts); err != nil {
			err = fmt.Errorf("syncing ticket %d: %w", ticket.ID, err)
			ph.step(err)
			errCh <- err
			return
		}
		ph.step(nil)
	})
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("sync: syncing open ticket", "error", err.Error())
		}
	}

	return stopped
}

func boardIDParam(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	var b strings.Builder
	for i, id := range ids {
		fmt.Fprintf(&b, "board/id = %d", id)
		if i < len(ids)-1 {
			b.WriteString(" OR ")
		}
	}

	return fmt.Sprintf("(%s)", b.String())
}
