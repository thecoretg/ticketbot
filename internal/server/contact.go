package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) ensureContactInStore(ctx context.Context, contactID int) (db.CwContact, error) {
	contact, err := cl.Queries.GetContact(ctx, contactID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cwContact, err := cl.CWClient.GetContact(contactID, nil)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("getting contact from cw: %w", err)
			}

			p := db.InsertContactParams{
				ID:        cwContact.ID,
				FirstName: cwContact.FirstName,
				LastName:  strToPtr(cwContact.LastName),
				CompanyID: intToPtr(cwContact.Company.ID),
			}

			contact, err = cl.Queries.InsertContact(ctx, p)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("inserting contact into db: %w", err)
			}
		} else {
			return db.CwContact{}, fmt.Errorf("getting contact from db: %w", err)
		}
	}

	return contact, nil
}
