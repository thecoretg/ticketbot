// Package transformer implements a configurable pipeline of "transformer" rules
// that mutate a Connectwise ticket (via the CW API) before the bot syncs it
// locally. Each action is a Transformer implementation registered by its action
// type. The pipeline is a no-op unless the feature is enabled and rules exist.
package transformer

import (
	"context"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// Exec bundles the Connectwise client and bot identity that transformers need to
// apply changes. It is passed to every Apply call so transformers stay stateless.
type Exec struct {
	CW                  *psa.Client
	BotMemberIdentifier string
}

// Change describes what a transformer did, for logging.
type Change struct {
	Applied bool   // false => transformer decided the ticket was already in the desired state
	Field   string // e.g. "summary", "note"
	From    string
	To      string
}

// Params is the marker interface for a transformer's typed parameters. Concrete
// params are plain structs whose templated string fields carry a `tmpl:"..."` tag.
type Params interface{ isParams() }

// Transformer is the extension point. Adding a new action = implement this
// interface and register it in newRegistry.
type Transformer interface {
	// ActionType is the stable key persisted in transformer_rule.action.
	ActionType() string

	// NewParams returns a zero-value params struct for this action. The engine
	// JSON-unmarshals the rule's config into it, then renders templated fields.
	NewParams() Params

	// Idempotent reports whether re-applying the action is safe. Non-idempotent
	// actions (e.g. adding a note) are guarded by run-once markers.
	Idempotent() bool

	// Apply mutates the ticket via the CW API. It receives the current ticket and
	// the rendered params, and returns a Change describing the result. Field-
	// mutating transformers should also update t in place so later rules in the
	// pipeline observe the change.
	Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error)
}

func newRegistry() map[string]Transformer {
	reg := map[string]Transformer{}
	for _, t := range []Transformer{
		UpdateSummary{},
		AddNote{},
	} {
		reg[t.ActionType()] = t
	}
	return reg
}
