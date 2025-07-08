package ticketbot

import (
	"context"
	"fmt"
	"tctg-automation/internal/ticketbot/db"
	"tctg-automation/pkg/connectwise"
)

func (s *server) getMostRecentNote(ctx context.Context, ticketID int) (int, error) {
	p := &connectwise.QueryParams{OrderBy: "_info/dateEntered desc", PageSize: 1000}
	n, err := s.cwClient.ListServiceTicketNotes(ctx, ticketID, p)
	if err != nil {
		return 0, checkCWError("listing ticket notes", "ticket", err, ticketID)
	}

	for _, note := range n {
		if note.Text != "" {
			return note.ID, nil
		}
	}

	return 0, nil
}

func (s *server) ensureTicketNoteExists(ctx context.Context, ticketID, noteID int) error {
	note, err := s.dbHandler.GetTicketNote(noteID)
	if err != nil {
		return fmt.Errorf("querying db for note: %w", err)
	}

	if note == nil {
		r, err := s.cwClient.GetServiceTicketNote(ctx, ticketID, noteID, nil)
		if err != nil {
			return checkCWError("getting ticket note", "ticket", err, noteID)
		}

		if r.Contact.ID != 0 {
			if err := s.ensureContactExists(ctx, r.Contact.ID); err != nil {
				return fmt.Errorf("ensuring contact exists for ticket note: %w", err)
			}
		}

		// TODO: check if member exists, if not, create it

		n := db.NewTicketNote(ticketID, noteID, r.Contact.ID, r.Member.ID, r.Text, r.DateCreated, r.InternalAnalysisFlag)
		if err := s.dbHandler.UpsertTicketNote(n); err != nil {
			return fmt.Errorf("inserting new ticket note into db: %w", err)
		}
	}

	return nil
}
