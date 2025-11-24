package inmem

import (
	"context"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/models"
)

type NotificationRepo struct {
	mu   sync.RWMutex
	data map[int]models.TicketNotification
	next int
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{
		data: make(map[int]models.TicketNotification),
		next: 1,
	}
}

func (p *NotificationRepo) WithTx(tx pgx.Tx) models.TicketNotificationRepository {
	return p
}

func (p *NotificationRepo) ListAll(ctx context.Context) ([]models.TicketNotification, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.TicketNotification
	for _, v := range p.data {
		out = append(out, v)
	}
	return out, nil
}

func (p *NotificationRepo) ListByNoteID(ctx context.Context, noteID int) ([]models.TicketNotification, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	var out []models.TicketNotification
	for _, v := range p.data {
		if v.TicketNoteID == &noteID {
			out = append(out, v)
		}
	}
	return out, nil
}

func (p *NotificationRepo) ExistsForTicket(ctx context.Context, ticketID int) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.TicketID == ticketID {
			return true, nil
		}
	}
	return false, nil
}

func (p *NotificationRepo) ExistsForNote(ctx context.Context, noteID int) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, v := range p.data {
		if v.TicketNoteID != nil && *v.TicketNoteID == noteID {
			return true, nil
		}
	}
	return false, nil
}

func (p *NotificationRepo) Get(ctx context.Context, id int) (models.TicketNotification, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	v, ok := p.data[id]
	if !ok {
		return models.TicketNotification{}, models.ErrNotificationNotFound
	}
	return v, nil
}

func (p *NotificationRepo) Insert(ctx context.Context, n models.TicketNotification) (models.TicketNotification, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if n.ID == 0 {
		n.ID = p.next
		p.next++
	}
	p.data[n.ID] = n
	return n, nil
}

func (p *NotificationRepo) Delete(ctx context.Context, id int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, ok := p.data[id]; !ok {
		return models.ErrNotificationNotFound
	}
	delete(p.data, id)
	return nil
}
