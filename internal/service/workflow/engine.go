package workflow

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

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
	// LearnedAPIMember is the member identifier ConnectWise attributed to a note the engine posted.
	LearnedAPIMember string
}

type Engine struct {
	CW CWClient
}

func NewEngine(cw CWClient) *Engine {
	return &Engine{CW: cw}
}

// Run evaluates the workflow's rules in order against the ticket. Notify actions are collected as
// intents; add_note actions are executed immediately (unless DryRun) and the ticket is re-fetched
// so later rules see the new state. Errors inside a rule or action are recorded, never returned;
// only a nil workflow or ticket is an error.
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

	changed := ticketdiff.FieldNames(in.Changes)
	doc := cwquery.NewDocument(res.Ticket, in.TriggerNote, changed)

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
			out := e.runAction(ctx, ref, i, a, in, res)
			res.Actions = append(res.Actions, out)
			if out.Kind == models.ActionAddNote && out.Result == ResultOK {
				doc = cwquery.NewDocument(res.Ticket, in.TriggerNote, changed)
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
			out.Result, out.Err = ResultError, errors.New("notify action has no settings")
			return out
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

	case models.ActionAddNote:
		if a.AddNote == nil {
			out.Result, out.Err = ResultError, errors.New("add_note action has no settings")
			return out
		}
		e.addNote(ctx, a.AddNote, in, res, &out)

	default:
		out.Result, out.Err = ResultError, fmt.Errorf("unknown action kind %q", a.Kind)
	}

	return out
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
