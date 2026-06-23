// Package workflow implements a configurable pipeline of "workflows" that run
// against a Connectwise ticket (via the CW API) before the bot syncs it locally.
// A workflow is scoped to a board, fires on a chosen ticket event when its
// optional nested condition tree matches, and then runs an ordered list of
// actions. Each action is an ActionHandler registered by its action type. The
// pipeline is a no-op unless the feature is enabled and workflows exist.
package workflow

import (
	"context"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// Exec bundles the clients and identity that actions need to apply changes. It
// is passed to every Apply call so actions stay stateless. LastNote is the
// ticket's most recent note at run time (nil when the ticket has none), used by
// the send_message ticket card.
type Exec struct {
	CW               *psa.Client
	Webex            repos.MessageSender
	Recips           repos.WebexRecipientRepository
	CWCompanyID      string
	MaxMessageLength int
	LastNote         *psa.ServiceTicketNote
}

// Change describes what an action did, for logging. SkipNotify signals that the
// downstream notifier should be suppressed for this ticket (set by a
// send_message action configured to skip further notifications).
type Change struct {
	Applied    bool   // false => action decided the ticket was already in the desired state
	Field      string // e.g. "summary", "note", "message"
	From       string
	To         string
	SkipNotify bool
}

// RunResult is what a workflow pipeline pass reports back: whether the downstream
// notifier should be skipped, a human-readable list of timeline events for the
// ticket journal, and whether the webhook was triggered by the bot's own prior
// edit (loop echo). BotTriggered is the authoritative "ignore this run" signal —
// it reflects the ticket's editor BEFORE this run's actions, unlike the post-sync
// ticket (which this run's own ticket_update/add_note actions would mark as the bot).
type RunResult struct {
	SkipNotify   bool
	BotTriggered bool
	Events       []models.JournalEvent
}

// Params is the marker interface for an action's typed parameters. Concrete
// params are plain structs whose templated string fields carry a `tmpl:"..."` tag.
type Params interface{ isParams() }

// ActionHandler is the extension point. Adding a new action = implement this
// interface and register it in newRegistry.
type ActionHandler interface {
	// ActionType is the stable key persisted in each workflow action's type.
	ActionType() string

	// NewParams returns a zero-value params struct for this action. The engine
	// JSON-unmarshals the action's config into it, then renders templated fields.
	NewParams() Params

	// Idempotent reports whether re-applying the action is safe. Non-idempotent
	// actions (e.g. adding a note, sending a message) are guarded by run-once markers.
	Idempotent() bool

	// Apply mutates the ticket (or performs a side effect) via the CW/Webex API. It
	// receives the current ticket and the rendered params, and returns a Change
	// describing the result. Field-mutating actions should also update t in place so
	// later actions and workflows observe the change.
	Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error)
}

func newRegistry() map[string]ActionHandler {
	reg := map[string]ActionHandler{}
	for _, a := range []ActionHandler{
		TicketUpdate{},
		AddNote{},
		SendMessage{},
		SkipNotifications{},
		AddResource{},
		AddEmailCc{},
	} {
		reg[a.ActionType()] = a
	}
	return reg
}
