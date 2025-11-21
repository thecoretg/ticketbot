package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type MemberRepo struct {
	mu   sync.RWMutex
	data map[int]models.Member
	next int
}

func NewMemberRepo(pool *pgxpool.Pool) *MemberRepo {
	return &MemberRepo{
		data: make(map[int]models.Member),
		next: 1,
	}
}

func (p *MemberRepo) WithTx(tx pgx.Tx) models.MemberRepository {
	return p
}

func (p *MemberRepo) List(ctx context.Context) ([]models.Member, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.Member
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *MemberRepo) Get(ctx context.Context, id int) (models.Member, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return models.Member{}, models.ErrMemberNotFound
	}
	return v, nil
}

func (p *MemberRepo) GetByIdentifier(ctx context.Context, identifier string) (models.Member, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.Identifier == identifier {
			return v, nil
		}
	}

	return models.Member{}, models.ErrMemberNotFound
}

func (p *MemberRepo) Upsert(ctx context.Context, m models.Member) (models.Member, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m.ID == 0 {
		m.ID = p.next
		p.next++
	}
	p.data[m.ID] = m
	return m, nil
}

func (p *MemberRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrMemberNotFound
	}
	delete(p.data, id)
	return nil
}
