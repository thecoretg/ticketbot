package db

import (
	"database/sql"
	"errors"
	"fmt"
	"tctg-automation/pkg/util"
	"time"
)

type TicketNote struct {
	ID        int       `db:"note_id"`
	TicketID  int       `db:"ticket_id"`
	ContactID *int      `db:"contact_id"`
	MemberID  *int      `db:"member_id"`
	Content   *string   `db:"content"`
	CreatedOn time.Time `db:"created_on"`
	Internal  bool      `db:"internal"`
}

func NewTicketNote(ticketID, noteID, contactID, memberID int, content string, createdOn time.Time, internal bool) *TicketNote {
	return &TicketNote{
		ID:        noteID,
		TicketID:  ticketID,
		ContactID: util.IntToPtr(contactID),
		MemberID:  util.IntToPtr(memberID),

		Content:   util.StrToPtr(content),
		CreatedOn: createdOn,
		Internal:  internal,
	}
}

func (h *Handler) GetTicketNote(noteID int) (*TicketNote, error) {
	n := &TicketNote{}
	if err := h.DB.Get(n, "SELECT * FROM ticket_note WHERE note_id = $1", noteID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting note by id: %w", err)
	}
	return n, nil
}

func (h *Handler) ListAllTicketNotes() ([]TicketNote, error) {
	var notes []TicketNote
	if err := h.DB.Select(&notes, "SELECT * FROM ticket_note"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing all ticket notes: %w", err)
	}
	return notes, nil
}

func (h *Handler) ListTicketNotes(ticketID int) ([]TicketNote, error) {
	var notes []TicketNote
	if err := h.DB.Select(&notes, "SELECT * FROM ticket_note WHERE ticket_id = $1", ticketID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing notes for ticket %d: %w", ticketID, err)
	}
	return notes, nil
}

func (h *Handler) UpsertTicketNote(n *TicketNote) error {
	_, err := h.DB.NamedExec(UpsertTicketNoteSQL(), n)
	if err != nil {
		return fmt.Errorf("inserting ticket note: %w", err)
	}
	return nil
}

func (h *Handler) DeleteTicketNote(noteID int) error {
	_, err := h.DB.Exec("DELETE FROM ticket_note WHERE ticket_id = $1", noteID)
	return err
}

func UpsertTicketNoteSQL() string {
	return `INSERT INTO ticket_note (note_id, ticket_id, contact_id, member_id, content, created_on, internal)
		VALUES (:note_id, :ticket_id, :contact_id, :member_id,:content, :created_on, :internal)
		ON CONFLICT (note_id) DO UPDATE SET
			ticket_id = EXCLUDED.ticket_id,
			contact_id = EXCLUDED.contact_id,
			member_id = EXCLUDED.member_id,
			content = EXCLUDED.content,
			created_on = EXCLUDED.created_on,
			internal = EXCLUDED.internal`
}
