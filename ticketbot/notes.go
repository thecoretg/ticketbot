package ticketbot

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
)

func (s *Server) getLatestNoteFromCW(ticketID int) (*connectwise.ServiceTicketNote, error) {
	note, err := s.cwClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note from connectwise: %w", err)
	}

	if note == nil {
		note = &connectwise.ServiceTicketNote{}
	}

	return note, nil
}

func (s *Server) ensureNoteInStore(ctx context.Context, cwData *cwData) (db.TicketNote, error) {
	note, err := s.queries.GetTicketNote(ctx, cwData.note.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			note, err = s.queries.InsertTicketNote(ctx, db.InsertTicketNoteParams{
				ID:       note.ID,
				TicketID: note.TicketID,
				Notified: false,
			})

			if err != nil {
				return db.TicketNote{}, fmt.Errorf("inserting ticket note into db: %w", err)
			}

		} else {
			return db.TicketNote{}, fmt.Errorf("getting note from store: %w", err)
		}
	}

	return note, nil
}

func (s *Server) setNotified(ctx context.Context, noteID int, notified bool) error {
	_, err := s.queries.SetNoteNotified(ctx, db.SetNoteNotifiedParams{
		ID:       noteID,
		Notified: notified,
	})

	if err != nil {
		return err
	}

	return nil
}
