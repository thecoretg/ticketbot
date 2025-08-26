package ticketbot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5"
	"github.com/thecoretg/ticketbot/db"
)

func (s *Server) ensureCompanyInStore(ctx context.Context, id int) (db.CwCompany, error) {
	company, err := s.Queries.GetCompany(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Debug("company not in store, attempting insert", "company_id", id)
			cwComp, err := s.CWClient.GetCompany(id, nil)
			if err != nil {
				return db.CwCompany{}, fmt.Errorf("getting company from cw: %w", err)
			}
			p := db.InsertCompanyParams{
				ID:   cwComp.Id,
				Name: cwComp.Name,
			}
			slog.Debug("created insert company params", "id", p.ID, "name", p.Name)

			company, err = s.Queries.InsertCompany(ctx, p)
			if err != nil {
				return db.CwCompany{}, fmt.Errorf("inserting company into db: %w", err)
			}
			slog.Info("inserted company into store", "company_id", company.ID, "company_name", company.Name)
			return company, nil
		} else {
			return db.CwCompany{}, fmt.Errorf("getting company from db: %w", err)
		}
	}

	slog.Debug("got existing company from store", "company_id", company.ID, "company_name", company.Name)
	return company, nil
}
