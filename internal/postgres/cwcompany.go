package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type CompanyRepo struct {
	queries *db.Queries
}

func NewCompanyRepo(pool *pgxpool.Pool) *CompanyRepo {
	return &CompanyRepo{
		queries: db.New(pool),
	}
}

func (p *CompanyRepo) WithTx(tx pgx.Tx) repos.CompanyRepository {
	return &CompanyRepo{
		queries: db.New(tx)}
}

func (p *CompanyRepo) List(ctx context.Context) ([]*models.Company, error) {
	dbs, err := p.queries.ListCompanies(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.Company
	for _, d := range dbs {
		b = append(b, companyFromPG(d))
	}

	return b, nil
}

func (p *CompanyRepo) Search(ctx context.Context, f models.CompanySearch) ([]*models.Company, error) {
	params := db.SearchCompaniesParams{Ids: f.IDs, Lim: int32(searchLimit(f.Limit))}
	if q := strings.TrimSpace(f.Query); q != "" {
		params.Search = &q
	}

	dbs, err := p.queries.SearchCompanies(ctx, params)
	if err != nil {
		return nil, err
	}

	out := make([]*models.Company, 0, len(dbs))
	for _, d := range dbs {
		out = append(out, companyFromPG(d))
	}

	return out, nil
}

// searchLimit caps typeahead result sizes.
func searchLimit(n int) int {
	switch {
	case n <= 0:
		return 25
	case n > 200:
		return 200
	}
	return n
}

func (p *CompanyRepo) Get(ctx context.Context, id int) (*models.Company, error) {
	d, err := p.queries.GetCompany(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrCompanyNotFound
		}
		return nil, err
	}

	return companyFromPG(d), nil
}

func (p *CompanyRepo) Upsert(ctx context.Context, b *models.Company) (*models.Company, error) {
	d, err := p.queries.UpsertCompany(ctx, companyToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return companyFromPG(d), nil
}

func (p *CompanyRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteCompany(ctx, id)
}

func (p *CompanyRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteCompany(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrCompanyNotFound
		}
		return err
	}

	return nil
}

func companyToUpsertParams(c *models.Company) db.UpsertCompanyParams {
	return db.UpsertCompanyParams{
		ID:   c.ID,
		Name: c.Name,
	}
}

func companyFromPG(pg *db.CwCompany) *models.Company {
	return &models.Company{
		ID:        pg.ID,
		Name:      pg.Name,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
