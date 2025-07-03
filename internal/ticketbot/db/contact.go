package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"tctg-automation/pkg/util"
)

type Contact struct {
	ID        int     `db:"contact_id"`
	FirstName string  `db:"first_name"`
	LastName  *string `db:"last_name"`
	CompanyID *int    `db:"company_id"`
}

func NewContact(id int, firstName, lastName string, companyID int) *Contact {
	return &Contact{
		ID:        id,
		FirstName: firstName,
		LastName:  util.StrToPtr(lastName),
		CompanyID: util.IntToPtr(companyID),
	}
}

func (h *Handler) GetContact(contactID int) (*Contact, error) {
	c := &Contact{}
	if err := h.DB.Get(c, "SELECT * FROM contact WHERE contact_id = $1", contactID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting contact by id: %w", err)
	}
	return c, nil
}

func (h *Handler) ListContacts() ([]Contact, error) {
	var contacts []Contact
	if err := h.DB.Select(&contacts, "SELECT * FROM contact"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing contacts: %w", err)
	}
	return contacts, nil
}

func (h *Handler) UpsertContact(c *Contact) error {
	_, err := h.DB.NamedExec(UpsertContactSQL(), c)
	if err != nil {
		return fmt.Errorf("inserting contact: %w", err)
	}

	ln := ""
	if c.LastName != nil {
		ln = *c.LastName
	}

	slog.Info("contact added or updated", "contact_id", c.ID, "first_name", c.FirstName, "last_name", ln)
	return nil
}

func (h *Handler) DeleteContact(contactID int) error {
	_, err := h.DB.Exec("DELETE FROM contact WHERE contact_id = $1", contactID)
	if err != nil {
		return err
	}
	slog.Info("contact deleted", "contact_id", contactID)
	return nil
}

func UpsertContactSQL() string {
	return `INSERT INTO contact (contact_id, first_name, last_name, company_id)
		VALUES (:contact_id, :first_name, :last_name, :company_id)
		ON CONFLICT (contact_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			company_id = EXCLUDED.company_id`
}
