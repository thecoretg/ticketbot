package models

import "time"

type SyncStatusResponse struct {
	Status bool `json:"status"`
	// Run is the running sync, or the last one since the app started; nil before the first.
	Run *SyncRun `json:"run"`
	// Nightly is the scheduled sync's record, nil before its first run.
	Nightly *ScheduledJob `json:"nightly,omitempty"`
}

type SyncPayload struct {
	WebexRecipients bool `json:"webex_recipients"`
	CWBoards        bool `json:"cw_boards"`
	// CWMembers syncs every ConnectWise member. CWCompanies and CWContacts refresh the ones
	// ticketbot has stored; the rest arrive with the tickets that reference them.
	CWMembers   bool  `json:"cw_members"`
	CWCompanies bool  `json:"cw_companies"`
	CWContacts  bool  `json:"cw_contacts"`
	CWTickets   bool  `json:"cw_tickets"`
	BoardIDs    []int `json:"board_ids"`
	// RunWorkflows runs each synced ticket's workflow when the sync finds a change, as a
	// webhook would. Off, the sync only refreshes the stored copy.
	RunWorkflows       bool `json:"run_workflows"`
	MaxConcurrentSyncs int  `json:"max_concurrent_syncs"`
}

// SyncRun is the progress of one sync. It lives in memory only: a restart forgets it.
type SyncRun struct {
	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
	StartedBy  string     `json:"started_by"`
	// CancelledAt is set when someone asks the run to stop; FinishedAt follows once every
	// phase has wound down.
	CancelledAt *time.Time  `json:"cancelled_at"`
	CancelledBy string      `json:"cancelled_by"`
	Options     SyncOptions `json:"options"`
	Phases      []SyncPhase `json:"phases"`
}

type SyncOptions struct {
	CWBoards        bool  `json:"cw_boards"`
	WebexRecipients bool  `json:"webex_recipients"`
	CWMembers       bool  `json:"cw_members"`
	CWCompanies     bool  `json:"cw_companies"`
	CWContacts      bool  `json:"cw_contacts"`
	CWTickets       bool  `json:"cw_tickets"`
	BoardIDs        []int `json:"board_ids"`
	RunWorkflows    bool  `json:"run_workflows"`
}

// ScheduledJob is the persisted record of a job that runs on a clock, such as the nightly sync.
type ScheduledJob struct {
	Name           string     `json:"name"`
	LastStartedAt  *time.Time `json:"last_started_at,omitempty"`
	LastFinishedAt *time.Time `json:"last_finished_at,omitempty"`
	LastError      *string    `json:"last_error,omitempty"`
}

type SyncPhaseState string

const (
	// SyncPhaseFetching means the phase has no total yet: it is still listing what to sync.
	SyncPhaseFetching SyncPhaseState = "fetching"
	SyncPhaseRunning  SyncPhaseState = "running"
	SyncPhaseDone     SyncPhaseState = "done"
	SyncPhaseFailed   SyncPhaseState = "failed"
	// SyncPhaseCancelled means the run was cancelled before this phase finished.
	SyncPhaseCancelled SyncPhaseState = "cancelled"
)

const (
	SyncPhaseBoards          = "boards"
	SyncPhaseWebexRecipients = "webex_recipients"
	SyncPhaseMembers         = "members"
	SyncPhaseCompanies       = "companies"
	SyncPhaseContacts        = "contacts"
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
