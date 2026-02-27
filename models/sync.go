package models

type SyncStatusResponse struct {
	Status bool `json:"status"`
}

type SyncPayload struct {
	WebexRecipients    bool  `json:"webex_recipients"`
	CWBoards           bool  `json:"cw_boards"`
	CWTickets          bool  `json:"cw_tickets"`
	BoardIDs           []int `json:"board_ids"`
	MaxConcurrentSyncs int   `json:"max_concurrent_syncs"`
}
