package models

import "time"

// WorkflowBundleVersion is the export file format version.
const WorkflowBundleVersion = 1

// WorkflowBundle is the portable form of one or more workflows: everything they reference that
// is local to a ticketbot database (Webex recipients, lists) travels with them, keyed by the
// source database id, so an import can create what is missing and rewrite the references.
// ConnectWise ids (boards, statuses, members, companies) are the same on every instance that
// talks to the same ConnectWise tenant and are exported as they are.
type WorkflowBundle struct {
	Version    int               `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	Recipients []BundleRecipient `json:"recipients"`
	Lists      []BundleList      `json:"lists"`
	Workflows  []Workflow        `json:"workflows"`
}

// BundleRecipient is a Webex room or person a notify node names. ID is the source database id
// the workflow nodes still carry; WebexID is what identifies it on import.
type BundleRecipient struct {
	ID      int                `json:"id"`
	WebexID string             `json:"webex_id"`
	Name    string             `json:"name"`
	Email   *string            `json:"email,omitempty"`
	Type    WebexRecipientType `json:"type"`
}

// BundleList is a list with its member ids. ID is the source database id conditions reference.
type BundleList struct {
	ID          int          `json:"id"`
	Name        string       `json:"name"`
	ItemType    ListItemType `json:"item_type"`
	Description string       `json:"description,omitempty"`
	Items       []int        `json:"items"`
}

// ImportOptions controls what an import does with a board that already has a workflow.
type ImportOptions struct {
	// Replace overwrites an existing workflow on the same board; otherwise it is skipped.
	Replace bool `json:"replace"`
}

// ImportReport says what an import did, workflow by workflow.
type ImportReport struct {
	RecipientsCreated []string             `json:"recipients_created"`
	ListsCreated      []string             `json:"lists_created"`
	Workflows         []ImportWorkflowItem `json:"workflows"`
	Warnings          []string             `json:"warnings"`
}

// ImportWorkflowItem is one workflow's outcome: created, replaced, skipped or error.
type ImportWorkflowItem struct {
	Name      string `json:"name"`
	BoardID   int    `json:"board_id"`
	BoardName string `json:"board_name,omitempty"`
	Result    string `json:"result"`
	Error     string `json:"error,omitempty"`
	ID        int    `json:"id,omitempty"`
}
