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

type MemberRepo struct {
	queries *db.Queries
}

func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{
		queries: db.New(pool),
	}
}

func (p *MemberRepo) WithTx(tx pgx.Tx) repos.MemberRepository {
	return &MemberRepo{
		queries: db.New(tx)}
}

func (p *MemberRepo) List(ctx context.Context) ([]*models.Member, error) {
	dm, err := p.queries.ListMembers(ctx)
	if err != nil {
		return nil, err
	}

	var b []*models.Member
	for _, d := range dm {
		b = append(b, memberFromPG(d))
	}

	return b, nil
}

func (p *MemberRepo) Get(ctx context.Context, id int) (*models.Member, error) {
	d, err := p.queries.GetMember(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrMemberNotFound
		}
		return nil, err
	}

	return memberFromPG(d), nil
}

func (p *MemberRepo) GetByIdentifier(ctx context.Context, identifier string) (*models.Member, error) {
	d, err := p.queries.GetMemberByIdentifier(ctx, identifier)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, models.ErrMemberNotFound
		}
		return nil, err
	}

	return memberFromPG(d), nil
}

func (p *MemberRepo) Upsert(ctx context.Context, b *models.Member) (*models.Member, error) {
	d, err := p.queries.UpsertMember(ctx, memberToUpsertParams(b))
	if err != nil {
		return nil, err
	}

	return memberFromPG(d), nil
}

func (p *MemberRepo) SoftDelete(ctx context.Context, id int) error {
	return p.queries.SoftDeleteMember(ctx, id)
}

func (p *MemberRepo) Delete(ctx context.Context, id int) error {
	if err := p.queries.DeleteMember(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.ErrMemberNotFound
		}
		return err
	}

	return nil
}

func memberToUpsertParams(m *models.Member) db.UpsertMemberParams {
	return db.UpsertMemberParams{
		ID:           m.ID,
		Identifier:   m.Identifier,
		FirstName:    m.FirstName,
		LastName:     m.LastName,
		PrimaryEmail: m.PrimaryEmail,
	}
}

func memberFromPG(pg *db.CwMember) *models.Member {
	return &models.Member{
		ID:           pg.ID,
		Identifier:   pg.Identifier,
		FirstName:    pg.FirstName,
		LastName:     pg.LastName,
		PrimaryEmail: pg.PrimaryEmail,
		UpdatedOn:    pg.UpdatedOn,
		AddedOn:      pg.AddedOn,
		Deleted:      pg.Deleted,
	}
}
