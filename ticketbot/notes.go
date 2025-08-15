package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
	"log/slog"
)

func (s *Server) getLatestNoteFromCW(ticketID int) (*connectwise.ServiceTicketNote, error) {
	note, err := s.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note from connectwise: %w", err)
	}

	if note == nil {
		note = &connectwise.ServiceTicketNote{}
	}

	return note, nil
}

func (s *Server) ensureNoteInStore(ctx context.Context, cwData *cwData, overrideNotify bool) (db.TicketNote, error) {
	note, err := s.Queries.GetTicketNote(ctx, cwData.note.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("note not found in store, attempting insert", "ticket_id", cwData.ticket.ID, "note_id", cwData.note.ID)
			note, err = s.Queries.InsertTicketNote(ctx, db.InsertTicketNoteParams{
				ID:       cwData.note.ID,
				TicketID: cwData.note.TicketId,
				Notified: overrideNotify,
				Member:   getMemberIdentifier(cwData),
				Contact:  getContactName(cwData),
			})

			if err != nil {
				return db.TicketNote{}, fmt.Errorf("inserting ticket note into db: %w", err)
			}

			sender := ""
			if note.Member != nil {
				sender = fmt.Sprintf("%s (member)", *note.Member)
			}

			if note.Contact != nil {
				sender = fmt.Sprintf("%s (contact)", *note.Contact)
			}

			slog.Info("inserted note into store", "ticket_id", cwData.ticket.ID, "note_id", cwData.note.ID, "sender", sender)
			return note, nil

		} else {
			return db.TicketNote{}, fmt.Errorf("getting note from store: %w", err)
		}
	}

	slog.Debug("note already in store", "ticket_id", cwData.ticket.ID, "note_id", cwData.note.ID)
	return note, nil
}

func (s *Server) setNotified(ctx context.Context, noteID int, notified bool) error {
	_, err := s.Queries.SetNoteNotified(ctx, db.SetNoteNotifiedParams{
		ID:       noteID,
		Notified: notified,
	})

	if err != nil {
		return err
	}

	return nil
}

func getMemberIdentifier(cwData *cwData) *string {
	if cwData.note.Member.Identifier != "" {
		return &cwData.note.Member.Identifier
	}

	return nil
}

func getContactName(cwData *cwData) *string {
	if cwData.note.Contact.Name != "" {
		return &cwData.note.Contact.Name
	}

	return nil
}
