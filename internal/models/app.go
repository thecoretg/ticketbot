package models

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

var ErrConfigNotFound = errors.New("config not found")

type Config struct {
	ID                 int  `json:"id"`
	Debug              bool `json:"debug"`
	AttemptNotify      bool `json:"attempt_notify"`
	MaxMessageLength   int  `json:"max_message_length"`
	MaxConcurrentSyncs int  `json:"max_concurrent_syncs"`
}

type ConfigRepository interface {
	WithTx(tx pgx.Tx) ConfigRepository
	Get(ctx context.Context) (Config, error)
	InsertDefault(ctx context.Context) (Config, error)
	Upsert(ctx context.Context, c Config) (Config, error)
}
