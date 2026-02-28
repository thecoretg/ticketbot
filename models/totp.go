package models

import (
	"errors"
	"time"
)

var ErrTOTPPendingNotFound = errors.New("totp pending token not found or expired")

type TOTPPending struct {
	ID        int
	UserID    int
	TokenHash []byte
	ExpiresAt time.Time
	CreatedOn time.Time
}

type TOTPRecoveryCode struct {
	ID       int
	UserID   int
	CodeHash []byte
	Used     bool
}
