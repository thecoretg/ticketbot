package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
)

func (s *Server) getLatestNoteFromCW(ticketID int) (*connectwise.ServiceTicketNote, error) {
	note, err := s.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note from connectwise: %w", err)
	}

	if note == nil {
		slog.Debug("no most recent note found", "ticket_id", ticketID)
		note = &connectwise.ServiceTicketNote{}
	}

	return note, nil
}

func (s *Server) ensureNoteInStore(ctx context.Context, cwData *cwData, overrideNotify bool) (db.CwTicketNote, error) {
	memberID, err := s.getMemberID(ctx, cwData)
	if err != nil {
		return db.CwTicketNote{}, fmt.Errorf("getting member data: %w", err)
	}

	contactID, err := s.getContactID(ctx, cwData)
	if err != nil {
		return db.CwTicketNote{}, fmt.Errorf("getting contact data: %w", err)
	}

	note, err := s.Queries.GetTicketNote(ctx, cwData.note.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("note not found in store, attempting insert", "ticket_id", cwData.ticket.ID, "note_id", cwData.note.ID)
			p := db.InsertTicketNoteParams{
				ID:        cwData.note.ID,
				TicketID:  cwData.note.TicketId,
				MemberID:  memberID,
				ContactID: contactID,
				Notified:  overrideNotify,
			}
			slog.Debug("created insert note params", "id", p.ID, "ticket_id", p.TicketID, "member_id", p.MemberID, "contact_id", p.ContactID, "notified", p.Notified)
			note, err = s.Queries.InsertTicketNote(ctx, p)

			if err != nil {
				return db.CwTicketNote{}, fmt.Errorf("inserting ticket note into db: %w", err)
			}

			slog.Info("inserted note into store", "ticket_id", cwData.ticket.ID, "note_id", cwData.note.ID)
			return note, nil

		} else {
			return db.CwTicketNote{}, fmt.Errorf("getting note from store: %w", err)
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

func (s *Server) getContactID(ctx context.Context, cwData *cwData) (*int, error) {
	if cwData.note.Contact.ID != 0 {
		c, err := s.ensureContactInStore(ctx, cwData.note.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact in store: %w", err)
		}

		return intToPtr(c.ID), nil
	}

	return nil, nil

}

func (s *Server) getMemberID(ctx context.Context, cwData *cwData) (*int, error) {
	if cwData.note.Member.ID != 0 {
		m, err := s.ensureMemberInStore(ctx, cwData.note.Member.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring member in store: %w", err)
		}

		return intToPtr(m.ID), nil
	}

	return nil, nil
}
