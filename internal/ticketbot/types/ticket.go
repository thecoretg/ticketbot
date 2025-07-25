package types

import "time"

type TimeDetails struct {
	UpdatedAt time.Time `json:"updated_at"`
}

type Ticket struct {
	ID           int    `json:"id"`
	Summary      string `json:"summary"`
	LatestNoteID int    `json:"latest_note_id"`
	UpdatedBy    string `json:"updated_by"`
	TimeDetails
}
