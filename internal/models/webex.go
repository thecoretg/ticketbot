package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrWebexRecipientNotFound = errors.New("webex room not found")

type WebexRecipient struct {
	ID           int                `json:"id"`
	WebexID      string             `json:"webex_id"`
	Name         string             `json:"name"`
	Email        *string            `json:"email"`
	Type         WebexRecipientType `json:"type"`
	LastActivity time.Time          `json:"last_activity"`
	CreatedOn    time.Time          `json:"created_on"`
	UpdatedOn    time.Time          `json:"updated_on"`
}

type WebexRecipientType string

const (
	RecipientTypeRoom   WebexRecipientType = "room"
	RecipientTypePerson WebexRecipientType = "person"
	// RecipientTypeUnknown WebexRecipientType = "unknown"
)

type WebexRecipientRepository interface {
	WithTx(tx pgx.Tx) WebexRecipientRepository
	List(ctx context.Context) ([]*WebexRecipient, error)
	ListRooms(ctx context.Context) ([]*WebexRecipient, error)
	ListPeople(ctx context.Context) ([]*WebexRecipient, error)
	ListByEmail(ctx context.Context, email string) ([]*WebexRecipient, error)
	Get(ctx context.Context, id int) (*WebexRecipient, error)
	GetByWebexID(ctx context.Context, webexID string) (*WebexRecipient, error)
	Upsert(ctx context.Context, r *WebexRecipient) (*WebexRecipient, error)
	Delete(ctx context.Context, id int) error
}
