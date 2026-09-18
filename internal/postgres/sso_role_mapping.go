package postgres

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/models"
)

type SSORoleMappingRepo struct {
	queries *db.Queries
}

func NewSSORoleMappingRepo(pool *pgxpool.Pool) *SSORoleMappingRepo {
	return &SSORoleMappingRepo{queries: db.New(pool)}
}

func (p *SSORoleMappingRepo) List(ctx context.Context) ([]*models.SSORoleMapping, error) {
	rows, err := p.queries.ListSSORoleMappings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*models.SSORoleMapping, 0, len(rows))
	for _, r := range rows {
		out = append(out, ssoRoleMappingFromPG(r))
	}
	return out, nil
}

func (p *SSORoleMappingRepo) Upsert(ctx context.Context, entraRole string, role models.Role) (*models.SSORoleMapping, error) {
	r, err := p.queries.UpsertSSORoleMapping(ctx, db.UpsertSSORoleMappingParams{EntraRole: entraRole, Role: string(role)})
	if err != nil {
		return nil, err
	}
	return ssoRoleMappingFromPG(r), nil
}

func (p *SSORoleMappingRepo) Delete(ctx context.Context, id int) error {
	return p.queries.DeleteSSORoleMapping(ctx, id)
}

func ssoRoleMappingFromPG(pg *db.SsoRoleMapping) *models.SSORoleMapping {
	return &models.SSORoleMapping{ID: pg.ID, EntraRole: pg.EntraRole, Role: models.Role(pg.Role)}
}
