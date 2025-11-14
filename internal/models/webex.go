package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrWebexRoomNotFound = errors.New("webex room not found")

type WebexRoom struct {
	ID        int       `json:"id"`
	WebexID   string    `json:"webex_id"`
	Name      string    `json:"name"`
	Type      string    `json:"type"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type WebexRoomRepository interface {
	WithTx(tx pgx.Tx) WebexRoomRepository
	List(ctx context.Context) ([]WebexRoom, error)
	Get(ctx context.Context, id int) (WebexRoom, error)
	GetByWebexID(ctx context.Context, webexID string) (WebexRoom, error)
	Upsert(ctx context.Context, r WebexRoom) (WebexRoom, error)
	Delete(ctx context.Context, id int) error
}
