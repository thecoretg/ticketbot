package models

import (
	"errors"
	"time"
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
)
