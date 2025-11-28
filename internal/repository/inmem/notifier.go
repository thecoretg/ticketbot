package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotifierRuleRepo struct {
	mu   sync.RWMutex
	data map[int]models.NotifierRule
	next int
}

func NewNotifierRuleRepo(pool *pgxpool.Pool) *NotifierRuleRepo {
	return &NotifierRuleRepo{
		data: make(map[int]models.NotifierRule),
		next: 1,
	}
}

func (p *NotifierRuleRepo) WithTx(tx pgx.Tx) models.NotifierRuleRepository {
	return p
}

func (p *NotifierRuleRepo) ListAll(ctx context.Context) ([]models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.NotifierRule
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *NotifierRuleRepo) ListByBoard(ctx context.Context, boardID int) ([]models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.NotifierRule
	for _, v := range p.data {
		if v.CwBoardID == boardID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (p *NotifierRuleRepo) ListByRoom(ctx context.Context, roomID int) ([]models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.NotifierRule
	for _, v := range p.data {
		if v.WebexRoomID == roomID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (p *NotifierRuleRepo) Get(ctx context.Context, id int) (*models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return nil, models.ErrNotifierNotFound
	}
	return &v, nil
}

func (p *NotifierRuleRepo) Exists(ctx context.Context, boardID, roomID int) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.CwBoardID == boardID && v.WebexRoomID == roomID {
			return true, nil
		}
	}
	return false, nil
}

func (p *NotifierRuleRepo) Insert(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	n.ID = p.next
	p.next++
	p.data[n.ID] = *n
	return n, nil
}

func (p *NotifierRuleRepo) Update(ctx context.Context, n *models.NotifierRule) (*models.NotifierRule, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[n.ID]; !ok {
		return nil, models.ErrNotifierNotFound
	}
	p.data[n.ID] = *n
	return n, nil
}

func (p *NotifierRuleRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrNotifierNotFound
	}
	delete(p.data, id)
	return nil
}
