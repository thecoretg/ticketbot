package workflow

import (
	"context"
	"fmt"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// SkipNotifications is a no-op action whose only effect is to signal the engine
// to suppress the default notifier for this ticket. Workflow-driven notifications
// (Send Notification actions) still fire. It carries no config.
type SkipNotifications struct{}

type SkipNotificationsParams struct{}

func (*SkipNotificationsParams) isParams() {}

func (SkipNotifications) ActionType() string { return models.WorkflowActionSkipNotifications }
func (SkipNotifications) NewParams() Params  { return &SkipNotificationsParams{} }
func (SkipNotifications) Idempotent() bool   { return true }

func (SkipNotifications) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	return Change{Applied: true, Field: "notifications", To: "skipped", SkipNotify: true}, nil
}

// AddResource adds a member (by identifier) to the ticket's resources list. CW
// stores resources as a comma-separated string of member identifiers, so this
// reads the current list, appends if absent, and patches the whole field. It is
// idempotent: a member already present is a no-op.
type AddResource struct{}

type AddResourceParams struct {
	MemberIdentifier string `json:"member_identifier"`
}

func (*AddResourceParams) isParams() {}

func (AddResource) ActionType() string { return models.WorkflowActionAddResource }
func (AddResource) NewParams() Params  { return &AddResourceParams{} }
func (AddResource) Idempotent() bool   { return true }

func (AddResource) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*AddResourceParams)
	member := strings.TrimSpace(pp.MemberIdentifier)
	if member == "" {
		return Change{Field: "resources"}, nil
	}

	existing := splitList(t.Resources)
	for _, r := range existing {
		if strings.EqualFold(r, member) {
			return Change{Field: "resources"}, nil // already a resource — no-op
		}
	}

	if x.Simulate {
		return Change{Applied: true, Field: "resources", To: member}, nil
	}

	updated, err := x.CW.PatchTicket(ctx, t.ID, []psa.PatchOp{{
		Op:    psa.Op(patchOpReplace),
		Path:  "resources",
		Value: strings.Join(append(existing, member), ","),
	}})
	if err != nil {
		return Change{}, fmt.Errorf("adding resource %q: %w", member, err)
	}
	*t = *updated

	return Change{Applied: true, Field: "resources", To: member}, nil
}

// AddEmailCc appends an email address to the ticket's automatic CC list and
// ensures CC delivery is enabled. CW stores the CC list as a single string; this
// reads it, appends if absent, and patches it back. Idempotent.
type AddEmailCc struct{}

type AddEmailCcParams struct {
	Email string `json:"email" tmpl:"email"`
}

func (*AddEmailCcParams) isParams() {}

func (AddEmailCc) ActionType() string { return models.WorkflowActionAddEmailCc }
func (AddEmailCc) NewParams() Params  { return &AddEmailCcParams{} }
func (AddEmailCc) Idempotent() bool   { return true }

func (AddEmailCc) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*AddEmailCcParams)
	email := strings.TrimSpace(pp.Email)
	if email == "" {
		return Change{Field: "email_cc"}, nil
	}

	existing := splitList(t.AutomaticEmailCc)
	for _, e := range existing {
		if strings.EqualFold(e, email) {
			return Change{Field: "email_cc"}, nil // already CC'd — no-op
		}
	}

	if x.Simulate {
		return Change{Applied: true, Field: "email_cc", To: email}, nil
	}

	updated, err := x.CW.PatchTicket(ctx, t.ID, []psa.PatchOp{
		{Op: psa.Op(patchOpReplace), Path: "automaticEmailCc", Value: strings.Join(append(existing, email), ";")},
		{Op: psa.Op(patchOpReplace), Path: "automaticEmailCcFlag", Value: true},
	})
	if err != nil {
		return Change{}, fmt.Errorf("adding email cc %q: %w", email, err)
	}
	*t = *updated

	return Change{Applied: true, Field: "email_cc", To: email}, nil
}

// splitList splits a CW delimited string (comma or semicolon separated) into
// trimmed, non-empty values.
func splitList(s string) []string {
	var out []string
	for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ';' }) {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
