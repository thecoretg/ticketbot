package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) ensureContactInStore(ctx context.Context, id int) (db.CwContact, error) {
	contact, err := cl.Queries.GetContact(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("contact not in store, attempting insert", "contact_id", id)
			cwContact, err := cl.CWClient.GetContact(id, nil)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("getting contact from cw: %w", err)
			}

			if cwContact.Company.ID != 0 {
				if _, err := cl.ensureCompanyInStore(ctx, cwContact.Company.ID); err != nil {
					return db.CwContact{}, fmt.Errorf("ensuring contact's company in store: %w", err)
				}
			}

			p := db.InsertContactParams{
				ID:        cwContact.ID,
				FirstName: cwContact.FirstName,
				LastName:  strToPtr(cwContact.LastName),
				CompanyID: intToPtr(cwContact.Company.ID),
			}
			slog.Debug("created insert contact params", "id", p.ID, "first_name", p.FirstName, "last_name", p.LastName, "company_id", p.CompanyID)

			contact, err = cl.Queries.InsertContact(ctx, p)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("inserting contact into db: %w", err)
			}
			slog.Debug("inserted contact into store", "contact_id", contact.ID, "first_name", contact.FirstName, "last_name", contact.LastName)
			return contact, nil
		} else {
			return db.CwContact{}, fmt.Errorf("getting contact from db: %w", err)
		}
	}

	slog.Debug("got existing contact from store", "contact_id", contact.ID, "first_name", contact.FirstName, "last_name", contact.LastName)
	return contact, nil
}
