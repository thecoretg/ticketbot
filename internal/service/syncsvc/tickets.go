package syncsvc

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/psa"
)

func (s *Service) SyncOpenTickets(ctx context.Context, boardIDs []int, maxSyncs int) error {
	start := time.Now()
	slog.Info("cwsvc: beginning ticket sync", "board_ids", boardIDs)
	con := "closedFlag = false"
	if len(boardIDs) > 0 {
		con += fmt.Sprintf(" AND %s", boardIDParam(boardIDs))
	}

	params := map[string]string{
		"pageSize":   "100",
		"conditions": con,
	}

	tix, err := s.CW.CWClient.ListTickets(params)
	if err != nil {
		return fmt.Errorf("getting open tickets from connectwise: %w", err)
	}
	slog.Info("cwsvc: open ticket sync: got open tickets from connectwise", "total_tickets", len(tix))
	sem := make(chan struct{}, maxSyncs)
	var wg sync.WaitGroup
	errCh := make(chan error, len(tix))

	for _, t := range tix {
		sem <- struct{}{}
		wg.Add(1)
		go func(ticket psa.Ticket) {
			defer func() { <-sem }()
			defer wg.Done()
			ft, err := s.CW.ProcessTicket(ctx, t.ID, "sync")
			if err != nil {
				errCh <- fmt.Errorf("error syncing ticket %d: %w", t.ID, err)
				return
			}

			if ft.LatestNote != nil && ft.LatestNote.ID != 0 {
				ti := ft.Ticket.ID
				i := ft.LatestNote.ID
				exists, err := s.Notifications.ExistsForNote(ctx, i)
				if err != nil {
					errCh <- fmt.Errorf("checking if notification for ticket %d note %d exists: %w", ti, i, err)
					return
				}

				if !exists {
					nt := models.TicketNotification{
						TicketID:     ti,
						TicketNoteID: &i,
						Skipped:      true,
					}

					nt, err = s.Notifications.Insert(ctx, nt)
					if err != nil {
						errCh <- fmt.Errorf("inserting skipped ticket notification for ticket %d note %d: %w", ti, i, err)
					}
					slog.Debug("ticket sync: inserted skipped notification", "notification_id", nt.ID, "ticket_id", ti, "note_id", i)
					return
				}
				slog.Debug("ticket sync: notification already exists for ticket note", "ticket_id", ti, "note_id", i)
			}
		}(t)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			slog.Error("cwsvc: syncing ticket", "error", err)
		}
	}
	slog.Info("cwsvc: syncing tickets complete", "took_time", time.Since(start))
	return nil
}

func boardIDParam(ids []int) string {
	if len(ids) == 0 {
		return ""
	}

	param := ""
	for i, id := range ids {
		param += fmt.Sprintf("board/id = %d", id)
		if i < len(ids)-1 {
			param += " OR "
		}
	}

	return fmt.Sprintf("(%s)", param)
}
