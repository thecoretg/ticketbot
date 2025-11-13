package note

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("ticket note not found")
)

type TicketNote struct {
	ID            int       `json:"id"`
	TicketID      int       `json:"ticket_id"`
	MemberID      *int      `json:"member_id"`
	ContactID     *int      `json:"contact_id"`
	Notified      bool      `json:"notified"`
	SkippedNotify bool      `json:"skipped_notify"`
	UpdatedOn     time.Time `json:"updated_on"`
	AddedOn       time.Time `json:"added_on"`
}

type Repository interface {
	ListByTicketID(ctx context.Context, ticketID int) ([]TicketNote, error)
	ListAll(ctx context.Context) ([]TicketNote, error)
	Get(ctx context.Context, id int) (TicketNote, error)
	Upsert(ctx context.Context, c TicketNote) (TicketNote, error)
	Delete(ctx context.Context, id int) error
}
