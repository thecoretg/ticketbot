package oldserver

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) ensureContactInStore(ctx context.Context, q *db.Queries, contactID int) (db.CwContact, error) {
	contact, err := q.GetContact(ctx, contactID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cwContact, err := cl.CWClient.GetContact(contactID, nil)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("getting contact from cw: %w", err)
			}

			var compID *int
			if cwContact.Company.ID != 0 {
				comp, err := cl.ensureCompanyInStore(ctx, q, cwContact.Company.ID)
				if err != nil {
					return db.CwContact{}, fmt.Errorf("ensuring contact's company is in db: %w", err)
				}

				compID = intToPtr(comp.ID)
			}

			p := db.InsertContactParams{
				ID:        cwContact.ID,
				FirstName: cwContact.FirstName,
				LastName:  strToPtr(cwContact.LastName),
				CompanyID: compID,
			}

			contact, err = q.InsertContact(ctx, p)
			if err != nil {
				return db.CwContact{}, fmt.Errorf("inserting contact into db: %w", err)
			}
		} else {
			return db.CwContact{}, fmt.Errorf("getting contact from db: %w", err)
		}
	}

	return contact, nil
}
