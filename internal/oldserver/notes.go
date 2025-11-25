package oldserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/external/psa"
)

func (cl *Client) getLatestNoteFromCW(ticketID int) (*psa.ServiceTicketNote, error) {
	note, err := cl.CWClient.GetMostRecentTicketNote(ticketID)
	if err != nil {
		return nil, fmt.Errorf("getting most recent note from connectwise: %w", err)
	}

	if note == nil {
		note = &psa.ServiceTicketNote{}
	}

	return note, nil
}

func (cl *Client) ensureNoteInStore(ctx context.Context, q *db.Queries, cwData *connectwiseData) (db.CwTicketNote, error) {
	memberID, err := cl.getMemberID(ctx, q, cwData)
	if err != nil {
		return db.CwTicketNote{}, fmt.Errorf("getting member data: %w", err)
	}

	contactID, err := cl.getContactID(ctx, q, cwData)
	if err != nil {
		return db.CwTicketNote{}, fmt.Errorf("getting contact data: %w", err)
	}

	note, err := q.GetTicketNote(ctx, cwData.note.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			p := db.UpsertTicketNoteParams{
				ID:        cwData.note.ID,
				TicketID:  cwData.note.TicketId,
				MemberID:  memberID,
				ContactID: contactID,
			}

			note, err = q.UpsertTicketNote(ctx, p)
			if err != nil {
				return db.CwTicketNote{}, fmt.Errorf("inserting ticket note into db: %w", err)
			}

			return note, nil

		} else {
			return db.CwTicketNote{}, fmt.Errorf("getting note from store: %w", err)
		}
	}

	return note, nil
}

func (cl *Client) setNotified(ctx context.Context, q *db.Queries, noteID int, notified bool) error {
	_, err := q.SetNoteNotified(ctx, db.SetNoteNotifiedParams{
		ID:       noteID,
		Notified: notified,
	})

	if err != nil {
		return err
	}

	return nil
}

func (cl *Client) setSkippedNotify(ctx context.Context, q *db.Queries, noteID int, skip bool) error {
	_, err := q.SetNoteSkippedNotify(ctx, db.SetNoteSkippedNotifyParams{
		ID:            noteID,
		SkippedNotify: skip,
	})

	if err != nil {
		return err
	}

	return nil
}

func (cl *Client) getContactID(ctx context.Context, q *db.Queries, cwData *connectwiseData) (*int, error) {
	if cwData.note.Contact.ID != 0 {
		c, err := cl.ensureContactInStore(ctx, q, cwData.note.Contact.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring contact in store: %w", err)
		}

		return intToPtr(c.ID), nil
	}

	return nil, nil

}

func (cl *Client) getMemberID(ctx context.Context, q *db.Queries, cwData *connectwiseData) (*int, error) {
	if cwData.note.Member.ID != 0 {
		m, err := cl.ensureMemberInStore(ctx, q, cwData.note.Member.ID)
		if err != nil {
			return nil, fmt.Errorf("ensuring member in store: %w", err)
		}

		return intToPtr(m.ID), nil
	}

	return nil, nil
}
