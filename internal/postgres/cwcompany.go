package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
	"github.com/thecoretg/ticketbot/internal/repos"
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
