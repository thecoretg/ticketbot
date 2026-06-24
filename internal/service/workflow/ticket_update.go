package workflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"text/template"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// Supported patch operation verbs.
const (
	patchOpReplace = "replace"
	patchOpRemove  = "remove"
)

// TicketUpdate applies a list of admin-configured operations to the live ticket
// in a single Connectwise PATCH call. Operations target an allowlisted set of
// fields (see updateFieldList); each replace value is a Go template rendered
// against the ticket. It is idempotent: ops whose target field already holds the
// desired value are dropped, and when no op remains no API call is made — so an
// action that re-fires on the bot's own edit is a no-op.
type TicketUpdate struct{}

// PatchOpConfig is one stored operation. Value is a single templated string
// (empty for remove); the backend coerces it to the Connectwise value shape
// based on the target field's kind.
type PatchOpConfig struct {
	Op    string `json:"op"`    // "replace" | "remove"
	Path  string `json:"path"`  // an allowlisted field path
	Value string `json:"value"` // templated string; ignored for remove
}

type TicketUpdateParams struct {
	Ops []PatchOpConfig `json:"ops"`
}

func (*TicketUpdateParams) isParams() {}

func (TicketUpdate) ActionType() string { return models.WorkflowActionTicketUpdate }
func (TicketUpdate) NewParams() Params  { return &TicketUpdateParams{} }
func (TicketUpdate) Idempotent() bool   { return true }

func (TicketUpdate) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*TicketUpdateParams)

	var (
		ops     []psa.PatchOp
		changed []string
	)
	for i, oc := range pp.Ops {
		f, ok := updateFieldByPath[oc.Path]
		if !ok {
			return Change{}, fmt.Errorf("op %d: unknown field %q", i+1, oc.Path)
		}

		switch oc.Op {
		case patchOpRemove:
			if f.Current(t) == "" {
				continue // already absent — idempotent no-op
			}
			ops = append(ops, psa.PatchOp{Op: psa.Op(patchOpRemove), Path: f.removePath()})
			changed = append(changed, oc.Path)

		case patchOpReplace:
			desired := oc.Value
			if f.MaxLen > 0 && len(desired) > f.MaxLen {
				slog.Warn("workflow: truncating value to Connectwise limit", "ticket_id", t.ID, "field", oc.Path, "limit", f.MaxLen)
				desired = desired[:f.MaxLen]
			}
			if desired == "" || strings.EqualFold(f.Current(t), desired) {
				continue // empty or already equal — idempotent no-op
			}
			val, err := f.buildValue(desired)
			if err != nil {
				return Change{}, fmt.Errorf("op %d (%s): %w", i+1, oc.Path, err)
			}
			ops = append(ops, psa.PatchOp{Op: psa.Op(patchOpReplace), Path: oc.Path, Value: val})
			changed = append(changed, oc.Path)

		default:
			return Change{}, fmt.Errorf("op %d: unsupported op %q", i+1, oc.Op)
		}
	}

	if len(ops) == 0 {
		return Change{Field: "ticket"}, nil // nothing to do
	}

	if x.Simulate {
		return Change{Applied: true, Field: strings.Join(changed, ","), To: fmt.Sprintf("%d op(s)", len(ops))}, nil
	}

	updated, err := x.CW.PatchTicket(ctx, t.ID, ops)
	if err != nil {
		return Change{}, fmt.Errorf("patching ticket: %w", err)
	}
	*t = *updated // refresh in-memory ticket so later actions observe the change

	return Change{Applied: true, Field: strings.Join(changed, ","), To: fmt.Sprintf("%d op(s)", len(ops))}, nil
}

// renderTemplates renders every replace op's value against the ticket. It is
// invoked by the engine in place of the generic reflect-based renderer because
// patch templates live in a slice of ops rather than tmpl-tagged struct fields.
func (p *TicketUpdateParams) renderTemplates(t *psa.Ticket) error {
	for i := range p.Ops {
		if p.Ops[i].Op == patchOpRemove {
			continue
		}
		rendered, err := renderTemplate(p.Ops[i].Value, t)
		if err != nil {
			return fmt.Errorf("op %d: %w", i+1, err)
		}
		p.Ops[i].Value = rendered
	}
	return nil
}

// validate checks every op references an allowlisted field, uses a supported
// verb, only removes removable fields, and (for replace) has a compilable
// template. Invoked at workflow-save time.
func (p *TicketUpdateParams) validate() error {
	if len(p.Ops) == 0 {
		return errors.New("ticket_update requires at least one operation")
	}
	for i, oc := range p.Ops {
		f, ok := updateFieldByPath[oc.Path]
		if !ok {
			return fmt.Errorf("op %d: unknown field %q", i+1, oc.Path)
		}
		switch oc.Op {
		case patchOpRemove:
			if !f.AllowRemove {
				return fmt.Errorf("op %d: field %q cannot be removed", i+1, oc.Path)
			}
		case patchOpReplace:
			if _, err := template.New("v").Option("missingkey=error").Parse(oc.Value); err != nil {
				return fmt.Errorf("op %d (%s): %w", i+1, oc.Path, err)
			}
		default:
			return fmt.Errorf("op %d: unsupported op %q", i+1, oc.Op)
		}
	}
	return nil
}

// checkDependencies enforces cross-op field dependencies: a field with DependsOn
// needs its parent set in the same workflow (contact→company, status→board), and
// a field with Requires forces its companion (board→status). A status op's board
// dependency is also satisfied by the workflow being board-scoped (hasBoard).
func (p *TicketUpdateParams) checkDependencies(hasBoard bool) error {
	present := map[string]bool{}
	for _, oc := range p.Ops {
		if oc.Op == patchOpRemove || strings.TrimSpace(oc.Value) != "" {
			present[oc.Path] = true
		}
	}

	for path := range present {
		f, ok := updateFieldByPath[path]
		if !ok {
			continue
		}
		if f.DependsOn != "" {
			satisfied := present[f.DependsOn]
			if path == "status" && hasBoard {
				satisfied = true
			}
			if !satisfied {
				return fmt.Errorf("%q requires %q to be set in the same workflow", path, f.DependsOn)
			}
		}
		if f.Requires != "" && !present[f.Requires] {
			return fmt.Errorf("%q also requires %q to be set in the same workflow", path, f.Requires)
		}
	}
	return nil
}
