package db

import (
	"database/sql"
	"errors"
	"fmt"
	"tctg-automation/pkg/util"
	"time"
)

type Ticket struct {
	ID         int        `db:"ticket_id"`
	Board      int        `db:"board_id"`
	Status     int        `db:"status_id"`
	Company    int        `db:"company_id"`
	Contact    *int       `db:"contact_id"`
	Summary    string     `db:"summary"`
	LatestNote *int       `db:"latest_note_id"`
	Owner      *int       `db:"owner_id"`
	Resources  *string    `db:"resources"`
	Created    time.Time  `db:"created_on"`
	Updated    time.Time  `db:"updated_on"`
	ClosedOn   *time.Time `db:"closed_on"`
	Closed     bool       `db:"closed"`
}

func NewTicket(ticketID, boardID, statusID, companyID, contactID, latestNoteID, ownerID int, summary, resources string, createdOn, updatedOn, closedOn time.Time, closed bool) *Ticket {
	return &Ticket{
		ID:         ticketID,
		Board:      boardID,
		Status:     statusID,
		Company:    companyID,
		Contact:    util.IntToPtr(contactID),
		LatestNote: util.IntToPtr(latestNoteID),
		Owner:      util.IntToPtr(ownerID),
		Summary:    summary,
		Resources:  util.StrToPtr(resources),
		Created:    createdOn,
		Updated:    updatedOn,
		ClosedOn:   util.TimeToPtr(closedOn),
		Closed:     closed,
	}
}

func (h *Handler) GetTicket(ticketID int) (*Ticket, error) {
	t := &Ticket{}
	if err := h.DB.Get(t, "SELECT * FROM ticket WHERE ticket_id = $1", ticketID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting ticket by id: %w", err)
	}

	return t, nil
}

func (h *Handler) ListTickets() ([]Ticket, error) {
	var tickets []Ticket
	if err := h.DB.Select(&tickets, "SELECT * FROM ticket"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing tickets: %w", err)
	}

	return tickets, nil
}

func (h *Handler) UpsertTicket(t *Ticket) error {
	_, err := h.DB.NamedExec(UpsertTicketSQL(), t)
	if err != nil {
		return fmt.Errorf("inserting ticket: %w", err)
	}
	return nil
}

func (h *Handler) DeleteTicket(ticketID int) error {
	_, err := h.DB.Exec("DELETE FROM ticket WHERE ticket_id = $1", ticketID)
	if err != nil {
		return err
	}

	return nil
}

func UpsertTicketSQL() string {
	return `INSERT INTO ticket (ticket_id, board_id, status_id, company_id, contact_id, latest_note_id, owner_id, summary, resources, created_on, updated_on, closed_on, closed)
		VALUES (:ticket_id, :board_id,:status_id, :company_id, :contact_id, :latest_note_id, :owner_id, :summary, :resources, :created_on, :updated_on, :closed_on, :closed)
		ON CONFLICT (ticket_id) DO UPDATE SET
			board_id = EXCLUDED.board_id,
			status_id = EXCLUDED.status_id,
			company_id = EXCLUDED.company_id,
			contact_id = EXCLUDED.contact_id,
			latest_note_id = EXCLUDED.latest_note_id,
			owner_id = EXCLUDED.owner_id,
			summary = EXCLUDED.summary,
			resources = EXCLUDED.resources,
			created_on = EXCLUDED.created_on,
			updated_on = EXCLUDED.updated_on,
			closed_on = EXCLUDED.closed_on,
			closed = EXCLUDED.closed`
}
