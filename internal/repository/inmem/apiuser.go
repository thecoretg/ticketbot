package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type APIUserRepo struct {
	mu   sync.RWMutex
	data map[int]models.APIUser
	next int
}

func NewAPIUserRepo(pool *pgxpool.Pool) *APIUserRepo {
	return &APIUserRepo{
		data: make(map[int]models.APIUser),
		next: 1,
	}
}

func (p *APIUserRepo) WithTx(tx pgx.Tx) models.APIUserRepository {
	return p
}

func (p *APIUserRepo) List(ctx context.Context) ([]models.APIUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.APIUser
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *APIUserRepo) Get(ctx context.Context, id int) (*models.APIUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return nil, models.ErrAPIUserNotFound
	}
	return &v, nil
}

func (p *APIUserRepo) Exists(ctx context.Context, email string) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.EmailAddress == email {
			return true, nil
		}
	}
	return false, nil
}

func (p *APIUserRepo) GetByEmail(ctx context.Context, email string) (*models.APIUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.EmailAddress == email {
			return &v, nil
		}
	}
	return nil, models.ErrAPIUserNotFound
}

func (p *APIUserRepo) Insert(ctx context.Context, email string) (*models.APIUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	u := models.APIUser{
		ID:           p.next,
		EmailAddress: email,
	}
	p.next++
	p.data[u.ID] = u
	return &u, nil
}

func (p *APIUserRepo) Update(ctx context.Context, u *models.APIUser) (*models.APIUser, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[u.ID]; !ok {
		return nil, models.ErrAPIUserNotFound
	}
	p.data[u.ID] = *u
	return u, nil
}

func (p *APIUserRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrAPIUserNotFound
	}
	delete(p.data, id)
	return nil
}
