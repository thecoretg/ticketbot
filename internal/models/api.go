package models

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

type CreateAPIKeyPayload struct {
	Email string `json:"email"`
}

type CreateAPIKeyResponse struct {
	Email string `json:"email"`
	Key   string `json:"key"`
}

type APIKey struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	KeyHash   []byte    `json:"key_hash"`
	CreatedOn time.Time `json:"created_on"`
	UpdatedOn time.Time `json:"updated_on"`
}

type APIKeyRepository interface {
	WithTx(tx pgx.Tx) APIKeyRepository
	List(ctx context.Context) ([]*APIKey, error)
	Get(ctx context.Context, id int) (*APIKey, error)
	Insert(ctx context.Context, a *APIKey) (*APIKey, error)
	Delete(ctx context.Context, id int) error
}

var ErrAPIUserNotFound = errors.New("api user not found")

type APIUser struct {
	ID           int       `json:"id"`
	EmailAddress string    `json:"email_address"`
	CreatedOn    time.Time `json:"created_on"`
	UpdatedOn    time.Time `json:"updated_on"`
}

type APIUserRepository interface {
	WithTx(tx pgx.Tx) APIUserRepository
	List(ctx context.Context) ([]*APIUser, error)
	Get(ctx context.Context, id int) (*APIUser, error)
	GetByEmail(ctx context.Context, email string) (*APIUser, error)
	Exists(ctx context.Context, email string) (bool, error)
	Insert(ctx context.Context, email string) (*APIUser, error)
	Delete(ctx context.Context, id int) error
}
