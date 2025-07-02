package db

import (
	"database/sql"
	"errors"
	"fmt"
	"tctg-automation/pkg/util"
)

type Member struct {
	ID         int     `db:"member_id"`
	Identifier string  `db:"identifier"`
	FirstName  string  `db:"first_name"`
	LastName   string  `db:"last_name"`
	Email      string  `db:"email"`
	Phone      *string `db:"phone"`
}

func NewMember(id int, identifier, firstName, lastName, email, phone string) *Member {
	return &Member{
		ID:         id,
		Identifier: identifier,
		FirstName:  firstName,
		LastName:   lastName,
		Email:      email,
		Phone:      util.StrToPtr(phone),
	}
}

func (h *Handler) GetMember(memberID int) (*Member, error) {
	m := &Member{}
	if err := h.DB.Get(m, "SELECT * FROM member WHERE member_id = $1", memberID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting member by id: %w", err)
	}
	return m, nil
}

func (h *Handler) ListMembers() ([]Member, error) {
	var members []Member
	if err := h.DB.Select(&members, "SELECT * FROM member"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing members: %w", err)
	}
	return members, nil
}

func (h *Handler) UpsertMember(m *Member) error {
	_, err := h.DB.NamedExec(UpsertMemberSQL(), m)
	if err != nil {
		return fmt.Errorf("inserting member: %w", err)
	}
	return nil
}

func (h *Handler) DeleteMember(memberID int) error {
	_, err := h.DB.Exec("DELETE FROM member WHERE member_id = $1", memberID)
	return err
}

func UpsertMemberSQL() string {
	return `INSERT INTO member (member_id, identifier, first_name, last_name, email, phone)
		VALUES (:member_id, :identifier, :first_name, :last_name, :email, :phone)
		ON CONFLICT (member_id) DO UPDATE SET
			identifier = EXCLUDED.identifier,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			email = EXCLUDED.email,
			phone = EXCLUDED.phone`
}
