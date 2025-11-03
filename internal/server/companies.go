package server

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/internal/db"
)

func (cl *Client) ensureCompanyInStore(ctx context.Context, companyID int) (db.CwCompany, error) {
	company, err := cl.Queries.GetCompany(ctx, companyID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			cwComp, err := cl.CWClient.GetCompany(companyID, nil)
			if err != nil {
				return db.CwCompany{}, fmt.Errorf("getting company from cw: %w", err)
			}
			p := db.UpsertCompanyParams{
				ID:   cwComp.Id,
				Name: cwComp.Name,
			}

			company, err = cl.Queries.UpsertCompany(ctx, p)
			if err != nil {
				return db.CwCompany{}, fmt.Errorf("inserting company into db: %w", err)
			}
		} else {
			return db.CwCompany{}, fmt.Errorf("getting company from db: %w", err)
		}
	}

	return company, nil
}
