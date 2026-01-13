package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/models"
)

type ContactRepo struct {
	queries *db.Queries
}

func NewContactRepo(pool *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{
		queries: db.New(pool),
	}
}

func (p *ContactRepo) WithTx(tx pgx.Tx) models.ContactRepository {
	return &ContactRepo{
		queries: db.New(tx)}
}

func (p *ContactRepo) List(ctx context.Context) ([]*models.Contact, error) {
	dbs, err := p.queries.ListContacts(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.Contact
	for _, d := range dbs {
		b = append(b, contactFromPG(d))
	}

	return b, nil
}

func (p *ContactRepo) Get(ctx context.Context, id int) (*models.Contact, error) {
	d, err := p.queries.GetContact(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrContactNotFound
		}
		return nil, err
	}

	return contactFromPG(d), nil
}

func (p *ContactRepo) Upsert(ctx context.Context, b *models.Contact) (*models.Contact, error) {
	d, err := p.queries.UpsertContact(ctx, contactToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return contactFromPG(d), nil
}

func (p *ContactRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteContact(ctx, id)
}

func (p *ContactRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteContact(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrContactNotFound
		}
		return err
	}

	return nil
}

func contactToUpsertParams(c *models.Contact) db.UpsertContactParams {
	return db.UpsertContactParams{
		ID:        c.ID,
		FirstName: c.FirstName,
		LastName:  c.LastName,
		CompanyID: c.CompanyID,
	}
}

func contactFromPG(pg *db.CwContact) *models.Contact {
	return &models.Contact{
		ID:        pg.ID,
		FirstName: pg.FirstName,
		LastName:  pg.LastName,
		CompanyID: pg.CompanyID,
		UpdatedOn: pg.UpdatedOn,
		AddedOn:   pg.AddedOn,
		Deleted:   pg.Deleted,
	}
}
