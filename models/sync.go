package models

import "time"

type SyncStatusResponse struct {
	Status bool `json:"status"`
	// Run is the running sync, or the last one since the app started; nil before the first.
	Run *SyncRun `json:"run"`
}

type SyncPayload struct {
	WebexRecipients    bool  `json:"webex_recipients"`
	CWBoards           bool  `json:"cw_boards"`
	CWTickets          bool  `json:"cw_tickets"`
	BoardIDs           []int `json:"board_ids"`
	MaxConcurrentSyncs int   `json:"max_concurrent_syncs"`
}

// SyncRun is the progress of one sync. It lives in memory only: a restart forgets it.
type SyncRun struct {
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt *time.Time  `json:"finished_at"`
	StartedBy  string      `json:"started_by"`
	Options    SyncOptions `json:"options"`
	Phases     []SyncPhase `json:"phases"`
}

type SyncOptions struct {
	CWBoards        bool  `json:"cw_boards"`
	WebexRecipients bool  `json:"webex_recipients"`
	CWTickets       bool  `json:"cw_tickets"`
	BoardIDs        []int `json:"board_ids"`
}

type SyncPhaseState string

const (
	// SyncPhaseFetching means the phase has no total yet: it is still listing what to sync.
	SyncPhaseFetching SyncPhaseState = "fetching"
	SyncPhaseRunning  SyncPhaseState = "running"
	SyncPhaseDone     SyncPhaseState = "done"
	SyncPhaseFailed   SyncPhaseState = "failed"
)

const (
	SyncPhaseBoards          = "boards"
	SyncPhaseWebexRecipients = "webex_recipients"
	SyncPhaseTickets         = "tickets"
)

// SyncPhase is one of the syncs a run started. Done counts items processed, failed ones
// included, so a finished phase always reaches Total. Errors keeps the first
// MaxSyncPhaseErrors messages; ErrorCount counts all of them.
type SyncPhase struct {
	Name       string         `json:"name"`
	State      SyncPhaseState `json:"state"`
	Label      string         `json:"label"`
	Done       int            `json:"done"`
	Total      int            `json:"total"`
	ErrorCount int            `json:"error_count"`
	Errors     []string       `json:"errors"`
}

const MaxSyncPhaseErrors = 20
