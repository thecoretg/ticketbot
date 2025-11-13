package api_key

import (
	"context"
	"errors"
	"time"
)

var (
	ErrNotFound = errors.New("api key not found")
)

type APIKey struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	KeyHash   []byte    `json:"key_hash"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type Repository interface {
	List(ctx context.Context) ([]APIKey, error)
	Get(ctx context.Context, id int) (APIKey, error)
	Insert(ctx context.Context, a APIKey) (APIKey, error)
	Delete(ctx context.Context, id int) error
}
