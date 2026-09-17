package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

// CWClient is the slice of the ConnectWise client the engine needs. *psa.Client satisfies it.
type CWClient interface {
	GetTicket(ctx context.Context, ticketID int, params map[string]string) (*psa.Ticket, error)
	GetMostRecentTicketNote(ctx context.Context, ticketID int) (*psa.ServiceTicketNote, error)
	PatchTicket(ctx context.Context, ticketID int, patchOps []psa.PatchOp) (*psa.Ticket, error)
	PostServiceTicketNote(ctx context.Context, ticketNote *psa.ServiceTicketNote, ticketID int) (*psa.ServiceTicketNote, error)
}

var _ CWClient = (*psa.Client)(nil)

// Input is everything the engine needs to run one workflow against one intake.
type Input struct {
	Ticket      *psa.Ticket
	TriggerNote *psa.ServiceTicketNote // most recent note at intake; nil when the ticket has none
	IsNew       bool
	NewNote     bool
	Changes     []models.FieldChange
	DryRun      bool // no ConnectWise writes; actions report would_run
}

// RuleRef identifies the rule an outcome belongs to.
type RuleRef struct {
	WorkflowID int
	RuleID     string
	RuleName   string
}

// NotifyIntent is a notify action that matched; the caller resolves recipients and sends.
type NotifyIntent struct {
	Rule        RuleRef
	ActionIndex int
	Target      models.NotifyAction
}

// Action results.
const (
	ResultOK       = "ok"
	ResultWouldRun = "would_run"
	ResultQueued   = "queued"
	ResultSkipped  = "skipped"
	ResultError    = "error"
)

type ActionOutcome struct {
	Rule   RuleRef
	Index  int
	Kind   models.ActionKind
	Result string
	Reason string
	Err    error
	Output map[string]any
}

type RuleOutcome struct {
	Rule    RuleRef
	Matched bool
	Skipped string // disabled | trigger
	Err     error
	Stopped bool
}

// Result is what one engine run produced.
type Result struct {
	// Ticket is the post-action ticket (re-fetched after any ConnectWise write).
	Ticket *psa.Ticket
	// LatestNote is the most recent note after actions ran; it may be a note the engine posted.
	LatestNote *psa.ServiceTicketNote
	// TriggerNote is unchanged from Input; notification text describes this note.
	TriggerNote *psa.ServiceTicketNote

	Rules      []RuleOutcome
	Actions    []ActionOutcome
	Notifies   []NotifyIntent
	Suppressed bool
	CWWrites   int
	// LearnedAPIMember is the member identifier ConnectWise attributed to a write the engine made.
	LearnedAPIMember string
}

type Engine struct {
	CW CWClient
}

func NewEngine(cw CWClient) *Engine {
	return &Engine{CW: cw}
}

// Run evaluates the workflow's rules in order against the ticket. Notify actions are collected as
// intents; ticket-mutating actions are executed immediately (unless DryRun) and the ticket is
// refreshed so later rules see the new state. Errors inside a rule or action are recorded, never
// returned; only a nil workflow or ticket is an error.
func (e *Engine) Run(ctx context.Context, wf *models.Workflow, in Input) (*Result, error) {
	if wf == nil {
		return nil, errors.New("nil workflow")
	}
	if in.Ticket == nil {
		return nil, errors.New("nil ticket")
	}

	res := &Result{
		Ticket:      in.Ticket,
		LatestNote:  in.TriggerNote,
		TriggerNote: in.TriggerNote,
	}

	changes := cwquery.Changes{Fields: in.Changes, NewNote: in.NewNote}
	doc := cwquery.NewDocument(res.Ticket, in.TriggerNote, changes)

	for _, r := range wf.Rules {
		ref := RuleRef{WorkflowID: wf.ID, RuleID: r.ID, RuleName: r.Name}

		if !r.Enabled {
			res.Rules = append(res.Rules, RuleOutcome{Rule: ref, Skipped: "disabled"})
			continue
		}
		if !r.Trigger.Matches(in.IsNew) {
			res.Rules = append(res.Rules, RuleOutcome{Rule: ref, Skipped: "trigger"})
			continue
		}

		q, err := cwquery.Compile(r.Condition)
		if err != nil {
			res.Rules = append(res.Rules, RuleOutcome{Rule: ref, Err: fmt.Errorf("compiling condition: %w", err)})
			continue
		}
		matched, err := q.Eval(doc)
		if err != nil {
			res.Rules = append(res.Rules, RuleOutcome{Rule: ref, Err: fmt.Errorf("evaluating condition: %w", err)})
			continue
		}
		if !matched {
			res.Rules = append(res.Rules, RuleOutcome{Rule: ref})
			continue
		}

		for i, a := range r.Actions {
			writes := res.CWWrites
			out := e.runAction(ctx, ref, i, a, in, res)
			res.Actions = append(res.Actions, out)
			if res.CWWrites != writes {
				doc = cwquery.NewDocument(res.Ticket, in.TriggerNote, changes)
			}
		}

		res.Rules = append(res.Rules, RuleOutcome{Rule: ref, Matched: true, Stopped: r.StopProcessing})
		if r.StopProcessing {
			break
		}
	}

	return res, nil
}

func (e *Engine) runAction(ctx context.Context, ref RuleRef, idx int, a models.Action, in Input, res *Result) ActionOutcome {
	out := ActionOutcome{Rule: ref, Index: idx, Kind: a.Kind}

	if !a.Enabled {
		out.Result, out.Reason = ResultSkipped, "disabled"
		return out
	}

	switch a.Kind {
	case models.ActionSkipNotify:
		res.Suppressed = true
		out.Result = ResultOK

	case models.ActionNotify:
		if a.Notify == nil {
			return missingSettings(out, "notify")
		}
		if res.Suppressed {
			out.Result, out.Reason = ResultSkipped, "suppressed"
			return out
		}
		res.Notifies = append(res.Notifies, NotifyIntent{Rule: ref, ActionIndex: idx, Target: *a.Notify})
		out.Result = ResultQueued
		out.Output = map[string]any{"target": string(a.Notify.Target)}
		if a.Notify.RecipientID != nil {
			out.Output["recipient_id"] = *a.Notify.RecipientID
		}
		if a.Notify.Message != "" {
			out.Output["custom_message"] = true
		}

	case models.ActionAddNote:
		if a.AddNote == nil {
			return missingSettings(out, "add_note")
		}
		e.addNote(ctx, a.AddNote, in, res, &out)

	case models.ActionSetStatus:
		if a.SetStatus == nil {
			return missingSettings(out, "set_status")
		}
		s := a.SetStatus
		out.Output = map[string]any{"status_id": s.StatusID, "status_name": s.StatusName}
		if res.Ticket.Status.ID == s.StatusID {
			out.Result, out.Reason = ResultSkipped, "already in this status"
			return out
		}
		e.patch(ctx, []psa.PatchOp{{Op: "replace", Path: "status/id", Value: s.StatusID}}, in, res, &out)

	case models.ActionSetPriority:
		if a.SetPriority == nil {
			return missingSettings(out, "set_priority")
		}
		p := a.SetPriority
		out.Output = map[string]any{"priority_id": p.PriorityID, "priority_name": p.PriorityName}
		if res.Ticket.Priority.ID == p.PriorityID {
			out.Result, out.Reason = ResultSkipped, "already at this priority"
			return out
		}
		e.patch(ctx, []psa.PatchOp{{Op: "replace", Path: "priority/id", Value: p.PriorityID}}, in, res, &out)

	case models.ActionSetOwner:
		if a.SetOwner == nil {
			return missingSettings(out, "set_owner")
		}
		o := a.SetOwner
		out.Output = map[string]any{"member_id": o.MemberID, "identifier": o.Identifier}
		if res.Ticket.Owner.ID == o.MemberID {
			out.Result, out.Reason = ResultSkipped, "already the owner"
			return out
		}
		e.patch(ctx, []psa.PatchOp{{Op: "replace", Path: "owner/id", Value: o.MemberID}}, in, res, &out)

	case models.ActionAddResource:
		if a.AddResource == nil {
			return missingSettings(out, "add_resource")
		}
		r := a.AddResource
		out.Output = map[string]any{"member_id": r.MemberID, "identifier": r.Identifier}
		if r.Identifier == "" {
			out.Result, out.Err = ResultError, errors.New("add_resource has no member identifier")
			return out
		}
		current := ticketdiff.Resources(res.Ticket.Resources)
		for _, id := range current {
			if strings.EqualFold(id, r.Identifier) {
				out.Result, out.Reason = ResultSkipped, "already a resource"
				return out
			}
		}
		joined := strings.Join(append(current, r.Identifier), ",")
		out.Output["resources"] = joined
		e.patch(ctx, []psa.PatchOp{{Op: "replace", Path: "resources", Value: joined}}, in, res, &out)

	case models.ActionPatch:
		if a.Patch == nil {
			return missingSettings(out, "patch")
		}
		ops, err := DecodePatchOps(a.Patch.Ops)
		if err != nil {
			out.Result, out.Err = ResultError, err
			return out
		}
		out.Output = map[string]any{"ops": ops}
		e.patch(ctx, ops, in, res, &out)

	default:
		out.Result, out.Err = ResultError, fmt.Errorf("unknown action kind %q", a.Kind)
	}

	return out
}

func missingSettings(out ActionOutcome, kind string) ActionOutcome {
	out.Result, out.Err = ResultError, fmt.Errorf("%s action has no settings", kind)
	return out
}

// DecodePatchOps parses a patch action's JSON into ConnectWise patch operations, rejecting
// anything that is not a non-empty array of {op, path[, value]} with a known op.
func DecodePatchOps(raw json.RawMessage) ([]psa.PatchOp, error) {
	if len(raw) == 0 {
		return nil, errors.New("patch ops are required")
	}

	var ops []psa.PatchOp
	if err := json.Unmarshal(raw, &ops); err != nil {
		return nil, fmt.Errorf("patch ops must be a JSON array of {op, path, value}: %w", err)
	}
	if len(ops) == 0 {
		return nil, errors.New("patch ops must contain at least one operation")
	}

	for i, op := range ops {
		switch op.Op {
		case "add", "replace", "remove":
		default:
			return nil, fmt.Errorf("op %d: op must be add, replace or remove (got %q)", i+1, op.Op)
		}
		if strings.TrimSpace(op.Path) == "" {
			return nil, fmt.Errorf("op %d: path is required", i+1)
		}
		if op.Op != "remove" && op.Value == nil {
			return nil, fmt.Errorf("op %d: value is required for %s", i+1, op.Op)
		}
	}

	return ops, nil
}

// patch applies ops to the ticket and refreshes the local copy from the response.
func (e *Engine) patch(ctx context.Context, ops []psa.PatchOp, in Input, res *Result, out *ActionOutcome) {
	if in.DryRun {
		out.Result = ResultWouldRun
		return
	}

	t, err := e.CW.PatchTicket(ctx, res.Ticket.ID, ops)
	if err != nil {
		out.Result, out.Err = ResultError, fmt.Errorf("patching ticket: %w", err)
		return
	}

	res.CWWrites++
	out.Result = ResultOK

	if t != nil && t.ID != 0 {
		res.Ticket = t
		if t.Info.UpdatedBy != "" {
			res.LearnedAPIMember = t.Info.UpdatedBy
		}
		return
	}

	if err := e.refetch(ctx, res); err != nil {
		// the write succeeded; a stale local copy is a warning, not an action failure
		slog.Warn("workflow: refetching ticket after patch", "ticket_id", res.Ticket.ID, "error", err.Error())
	}
}

func (e *Engine) addNote(ctx context.Context, n *models.AddNoteAction, in Input, res *Result, out *ActionOutcome) {
	out.Output = map[string]any{
		"text":       n.Text,
		"internal":   n.Internal,
		"discussion": n.Discussion,
		"resolution": n.Resolution,
	}

	if in.DryRun {
		out.Result = ResultWouldRun
		return
	}

	note := &psa.ServiceTicketNote{
		Text:                  n.Text,
		InternalAnalysisFlag:  n.Internal,
		DetailDescriptionFlag: n.Discussion,
		ResolutionFlag:        n.Resolution,
	}

	posted, err := e.CW.PostServiceTicketNote(ctx, note, res.Ticket.ID)
	if err != nil {
		out.Result, out.Err = ResultError, fmt.Errorf("posting note: %w", err)
		return
	}

	res.CWWrites++
	out.Result = ResultOK
	if posted != nil {
		out.Output["note_id"] = posted.ID
		res.LearnedAPIMember = posted.Member.Identifier
		res.LatestNote = posted
	}

	if err := e.refetch(ctx, res); err != nil {
		// the write succeeded; a stale local copy is a warning, not an action failure
		slog.Warn("workflow: refetching ticket after note", "ticket_id", res.Ticket.ID, "error", err.Error())
	}
}

// refetch reloads the ticket and its most recent note after a ConnectWise write.
func (e *Engine) refetch(ctx context.Context, res *Result) error {
	t, err := e.CW.GetTicket(ctx, res.Ticket.ID, nil)
	if err != nil {
		return fmt.Errorf("getting ticket: %w", err)
	}
	if t != nil {
		res.Ticket = t
	}

	n, err := e.CW.GetMostRecentTicketNote(ctx, res.Ticket.ID)
	if err != nil && !errors.Is(err, psa.ErrNotFound) {
		return fmt.Errorf("getting most recent note: %w", err)
	}
	if n != nil {
		res.LatestNote = n
	}

	return nil
}
