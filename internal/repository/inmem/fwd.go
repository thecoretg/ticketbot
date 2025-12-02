package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type UserForwardRepo struct {
	mu   sync.RWMutex
	data map[int]models.NotifierForward
	next int
}

func NewUserForwardRepo(pool *pgxpool.Pool) *UserForwardRepo {
	return &UserForwardRepo{
		data: make(map[int]models.NotifierForward),
		next: 1,
	}
}

func (p *UserForwardRepo) WithTx(tx pgx.Tx) models.NotifierForwardRepository {
	return p
}

func (p *UserForwardRepo) ListAll(ctx context.Context) ([]models.NotifierForward, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.NotifierForward
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *UserForwardRepo) ListBySourceRoomID(ctx context.Context, id int) ([]models.NotifierForward, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.NotifierForward
	for _, v := range p.data {
		if v.SourceID == id {
			out = append(out, v)
		}
	}
	return out, nil
}

func (p *UserForwardRepo) Get(ctx context.Context, id int) (models.NotifierForward, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return models.NotifierForward{}, models.ErrUserForwardNotFound
	}
	return v, nil
}

func (p *UserForwardRepo) Insert(ctx context.Context, b models.NotifierForward) (models.NotifierForward, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if b.ID == 0 {
		b.ID = p.next
		p.next++
	}
	p.data[b.ID] = b
	return b, nil
}

func (p *UserForwardRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrUserForwardNotFound
	}
	delete(p.data, id)
	return nil
}
