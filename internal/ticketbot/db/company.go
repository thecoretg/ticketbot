package db

import (
	"database/sql"
	"errors"
	"fmt"
)

type Company struct {
	ID   int    `db:"company_id"`
	Name string `db:"company_name"`
}

func NewCompany(id int, name string) *Company {
	return &Company{
		ID:   id,
		Name: name,
	}
}

func (h *Handler) GetCompany(companyID int) (*Company, error) {
	c := &Company{}
	if err := h.DB.Get(c, "SELECT * FROM company WHERE company_id = $1", companyID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("getting company by id: %w", err)
	}
	return c, nil
}

func (h *Handler) ListCompanies() ([]Company, error) {
	var companies []Company
	if err := h.DB.Select(&companies, "SELECT * FROM company"); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("listing companies: %w", err)
	}
	return companies, nil
}

func (h *Handler) UpsertCompany(c *Company) error {
	_, err := h.DB.NamedExec(UpsertCompanySQL(), c)
	if err != nil {
		return fmt.Errorf("inserting company: %w", err)
	}
	return nil
}

func (h *Handler) DeleteCompany(companyID int) error {
	_, err := h.DB.Exec("DELETE FROM company WHERE company_id = $1", companyID)
	return err
}

func UpsertCompanySQL() string {
	return `INSERT INTO company (company_id, company_name)
		VALUES (:company_id, :company_name)
		ON CONFLICT (company_id) DO UPDATE SET
			company_name = EXCLUDED.company_name`
}
