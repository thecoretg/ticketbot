package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type WebexRecipientRepo struct {
	mu   sync.RWMutex
	data map[int]models.WebexRecipient
	next int
}

func NewWebexRecipientRepo(pool *pgxpool.Pool) *WebexRecipientRepo {
	return &WebexRecipientRepo{
		data: make(map[int]models.WebexRecipient),
		next: 1,
	}
}

func (p *WebexRecipientRepo) WithTx(tx pgx.Tx) models.WebexRecipientRepository {
	return p
}

func (p *WebexRecipientRepo) List(ctx context.Context) ([]models.WebexRecipient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.WebexRecipient
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *WebexRecipientRepo) ListByEmail(ctx context.Context, email string) ([]models.WebexRecipient, error) {
	p.mu.Lock()
	defer p.mu.Lock()

	var rms []models.WebexRecipient
	for _, v := range p.data {
		if v.Email != nil && *v.Email == email {
			rms = append(rms, v)
		}
	}

	return rms, nil
}

func (p *WebexRecipientRepo) Get(ctx context.Context, id int) (models.WebexRecipient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return models.WebexRecipient{}, models.ErrWebexRecipientNotFound
	}
	return v, nil
}

func (p *WebexRecipientRepo) GetByWebexID(ctx context.Context, webexID string) (models.WebexRecipient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.WebexID == webexID {
			return v, nil
		}
	}
	return models.WebexRecipient{}, models.ErrWebexRecipientNotFound
}

func (p *WebexRecipientRepo) Upsert(ctx context.Context, r models.WebexRecipient) (models.WebexRecipient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if r.ID == 0 {
		r.ID = p.next
		p.next++
	}
	p.data[r.ID] = r
	return r, nil
}

func (p *WebexRecipientRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrWebexRecipientNotFound
	}
	delete(p.data, id)
	return nil
}
