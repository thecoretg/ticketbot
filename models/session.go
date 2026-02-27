package models

import (
	"errors"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID        int
	UserID    int
	TokenHash []byte
	ExpiresAt time.Time
	CreatedOn time.Time
}
