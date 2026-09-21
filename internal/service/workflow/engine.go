package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
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

// ListLoader supplies admin-list memberships for `in list` conditions. A nil loader means no
// lists: every `in list` is false and every `not in list` is true.
type ListLoader interface {
	Memberships(ctx context.Context) (cwquery.Lists, error)
}

// LoadEnv builds the evaluation environment for exprs. It only hits the loader when at least one
// expression references a list.
func LoadEnv(ctx context.Context, loader ListLoader, exprs ...cwquery.Expr) (cwquery.Env, error) {
	if loader == nil {
		return cwquery.Env{}, nil
	}
	for _, e := range exprs {
		if len(cwquery.ListRefs(e)) > 0 {
			lists, err := loader.Memberships(ctx)
			if err != nil {
				return cwquery.Env{}, err
			}
			return cwquery.Env{Lists: lists}, nil
		}
	}
	return cwquery.Env{}, nil
}

// MaxSteps caps the nodes one run may visit. Validation rejects cycles, but the engine must not
// trust the document it is handed.
const MaxSteps = 500

// Input is everything the engine needs to run one workflow against one intake.
type Input struct {
	Ticket      *psa.Ticket
	TriggerNote *psa.ServiceTicketNote // most recent note at intake; nil when the ticket has none
	IsNew       bool
	NewNote     bool
	Changes     []models.FieldChange
	DryRun      bool // no ConnectWise writes; actions report would_run
}

// StepRef identifies the node an outcome belongs to.
type StepRef struct {
	WorkflowID int
	NodeID     string
	Title      string
}

// NotifyIntent is a notify node that ran; the caller resolves recipients and sends.
type NotifyIntent struct {
	Step   StepRef
	Target models.NotifyAction
}

// Action results.
const (
	ResultOK       = "ok"
	ResultWouldRun = "would_run"
	ResultQueued   = "queued"
	ResultSkipped  = "skipped"
	ResultError    = "error"
)

// Step skip reasons.
const (
	SkippedDisabled = "disabled"
	SkippedJoined   = "joined" // another trigger's walk already ran this node
)

type ActionOutcome struct {
	Step   StepRef
	No     int // the step number this action ran as
	Kind   models.ActionKind
	Result string
	Reason string
	Err    error
	Output map[string]any
}

// Payload is the action in its ticket-history form.
func (a ActionOutcome) Payload() models.ActionPayload {
	p := models.ActionPayload{
		NodeID: a.Step.NodeID,
		Title:  a.Step.Title,
		No:     a.No,
		Kind:   string(a.Kind),
		Result: a.Result,
		Reason: a.Reason,
		Output: a.Output,
	}
	if a.Err != nil {
		p.Error = a.Err.Error()
	}
	return p
}

// StepOutcome records one node a walk visited.
type StepOutcome struct {
	Step    StepRef
	No      int // 1-based position in the run
	Kind    models.NodeKind
	Trigger string      // node id of the trigger whose walk this is
	Via     string      // edge id the walk arrived by; empty for the trigger itself
	Port    models.Port // output the walk left by; empty when the walk ended here
	Matched *bool       // if nodes: the condition's verdict
	Skipped string      // SkippedDisabled | SkippedJoined
	Err     error
}

// Payload is the step in its ticket-history form.
func (s StepOutcome) Payload() models.StepPayload {
	p := models.StepPayload{
		NodeID:  s.Step.NodeID,
		Title:   s.Step.Title,
		No:      s.No,
		Kind:    s.Kind,
		Trigger: s.Trigger,
		Via:     s.Via,
		Port:    s.Port,
		Matched: s.Matched,
		Skipped: s.Skipped,
	}
	if s.Err != nil {
		p.Error = s.Err.Error()
	}
	return p
}

// Result is what one engine run produced.
type Result struct {
	// Ticket is the post-action ticket (re-fetched after any ConnectWise write).
	Ticket *psa.Ticket
	// LatestNote is the most recent note after actions ran; it may be a note the engine posted.
	LatestNote *psa.ServiceTicketNote
	// TriggerNote is unchanged from Input; notification text describes this note.
	TriggerNote *psa.ServiceTicketNote

	// Event is what the intake raised; Steps is empty when no trigger listens for it.
	Event    models.TriggerEvent
	Steps    []StepOutcome
	Actions  []ActionOutcome
	Notifies []NotifyIntent
	CWWrites int
	// LearnedAPIMember is the member identifier ConnectWise attributed to a write the engine made.
	LearnedAPIMember string
}

type Engine struct {
	CW CWClient
	// Lists is optional; set it to make `in list` conditions see admin lists.
	Lists ListLoader
}

func NewEngine(cw CWClient) *Engine {
	return &Engine{CW: cw}
}

// run is the mutable state of one engine run.
type run struct {
	wf      *models.Workflow
	in      Input
	res     *Result
	nodes   map[string]*models.Node
	out     map[string]map[models.Port]*models.Edge // from node id → port → edge
	changes cwquery.Changes
	doc     map[string]any
	visited map[string]bool
	env     cwquery.Env
	envOK   bool
}

// Run walks the workflow graph for the ticket. Every enabled trigger listening for the intake's
// event starts a walk, in canvas order; each walk follows the port its nodes select until a port
// has no wire, a node another walk already ran is reached, or the step cap trips. Notify nodes are
// collected as intents; ticket-mutating nodes are executed immediately (unless DryRun) and the
// ticket is refreshed so later nodes see the new state. Errors inside a node are recorded, never
// returned; only a nil workflow or ticket is an error.
func (e *Engine) Run(ctx context.Context, wf *models.Workflow, in Input) (*Result, error) {
	if wf == nil {
		return nil, errors.New("nil workflow")
	}
	if in.Ticket == nil {
		return nil, errors.New("nil ticket")
	}

	r := &run{
		wf: wf,
		in: in,
		res: &Result{
			Ticket:      in.Ticket,
			LatestNote:  in.TriggerNote,
			TriggerNote: in.TriggerNote,
			Event:       models.EventFor(in.IsNew),
		},
		nodes:   make(map[string]*models.Node, len(wf.Nodes)),
		out:     make(map[string]map[models.Port]*models.Edge, len(wf.Nodes)),
		changes: cwquery.Changes{Fields: in.Changes, NewNote: in.NewNote, IsNew: in.IsNew},
		visited: make(map[string]bool, len(wf.Nodes)),
	}
	for i := range wf.Nodes {
		n := &wf.Nodes[i]
		r.nodes[n.ID] = n
	}
	for i := range wf.Edges {
		ed := &wf.Edges[i]
		if r.out[ed.From] == nil {
			r.out[ed.From] = map[models.Port]*models.Edge{}
		}
		if _, dup := r.out[ed.From][ed.Port]; !dup {
			r.out[ed.From][ed.Port] = ed
		}
	}
	r.doc = cwquery.NewDocument(r.res.Ticket, in.TriggerNote, r.changes)

	for _, t := range r.triggers() {
		e.walk(ctx, r, t)
		if len(r.res.Steps) >= MaxSteps {
			break
		}
	}

	return r.res, nil
}

// triggers lists the enabled trigger nodes listening for the event, left to right then top to
// bottom as they sit on the canvas.
func (r *run) triggers() []*models.Node {
	var out []*models.Node
	for _, n := range r.nodes {
		if n.Kind == models.NodeTrigger && n.Enabled && n.Listens(r.res.Event) {
			out = append(out, n)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.X != b.X {
			return a.X < b.X
		}
		if a.Y != b.Y {
			return a.Y < b.Y
		}
		return a.ID < b.ID
	})
	return out
}

func (e *Engine) walk(ctx context.Context, r *run, trigger *models.Node) {
	cur, via := trigger, ""
	suppressed := false // a skip_notify on this walk silences the notifies after it

	for cur != nil && len(r.res.Steps) < MaxSteps {
		ref := StepRef{WorkflowID: r.wf.ID, NodeID: cur.ID, Title: cur.Title}
		st := StepOutcome{Step: ref, No: len(r.res.Steps) + 1, Kind: cur.Kind, Trigger: trigger.ID, Via: via}

		if r.visited[cur.ID] {
			st.Skipped = SkippedJoined
			r.res.Steps = append(r.res.Steps, st)
			return
		}
		r.visited[cur.ID] = true

		var port models.Port
		switch {
		case cur.Kind == models.NodeTrigger:
			port = models.PortOut

		case cur.Kind == models.NodeIf:
			port = models.PortNo
			if !cur.Enabled {
				st.Skipped = SkippedDisabled
				break
			}
			matched, err := e.evalCondition(ctx, r, cur.Condition)
			if err != nil {
				st.Err = err
				break
			}
			st.Matched = &matched
			if matched {
				port = models.PortYes
			}

		case cur.Kind.IsAction():
			port = models.PortOut
			if !cur.Enabled {
				st.Skipped = SkippedDisabled
			}
			writes := r.res.CWWrites
			out := e.runAction(ctx, ref, st.No, cur.Action(), r, &suppressed)
			r.res.Actions = append(r.res.Actions, out)
			if r.res.CWWrites != writes {
				r.doc = cwquery.NewDocument(r.res.Ticket, r.in.TriggerNote, r.changes)
			}

		default:
			st.Err = fmt.Errorf("unknown node kind %q", cur.Kind)
		}

		next := r.out[cur.ID][port]
		if next == nil {
			r.res.Steps = append(r.res.Steps, st)
			return
		}
		st.Port = port
		r.res.Steps = append(r.res.Steps, st)
		cur, via = r.nodes[next.To], next.ID
	}
}

// evalCondition compiles and evaluates an if node's condition against the current document,
// loading list memberships the first time a condition needs them.
func (e *Engine) evalCondition(ctx context.Context, r *run, condition string) (bool, error) {
	q, err := cwquery.Compile(condition)
	if err != nil {
		return false, fmt.Errorf("compiling condition: %w", err)
	}
	if !r.envOK && len(cwquery.ListRefs(q.Expr)) > 0 {
		env, err := LoadEnv(ctx, e.Lists, q.Expr)
		if err != nil {
			return false, fmt.Errorf("loading lists: %w", err)
		}
		r.env, r.envOK = env, true
	}
	matched, err := q.EvalEnv(r.doc, r.env)
	if err != nil {
		return false, fmt.Errorf("evaluating condition: %w", err)
	}
	return matched, nil
}

func (e *Engine) runAction(ctx context.Context, ref StepRef, no int, a models.Action, r *run, suppressed *bool) ActionOutcome {
	in, res := r.in, r.res
	out := ActionOutcome{Step: ref, No: no, Kind: a.Kind}

	if !a.Enabled {
		out.Result, out.Reason = ResultSkipped, SkippedDisabled
		return out
	}

	switch a.Kind {
	case models.ActionSkipNotify:
		*suppressed = true
		out.Result = ResultOK

	case models.ActionNotify:
		if a.Notify == nil {
			return missingSettings(out, "notify")
		}
		if *suppressed {
			out.Result, out.Reason = ResultSkipped, "suppressed"
			return out
		}
		res.Notifies = append(res.Notifies, NotifyIntent{Step: ref, Target: *a.Notify})
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
		ar := a.AddResource
		out.Output = map[string]any{"member_id": ar.MemberID, "identifier": ar.Identifier}
		if ar.Identifier == "" {
			out.Result, out.Err = ResultError, errors.New("add_resource has no member identifier")
			return out
		}
		current := ticketdiff.Resources(res.Ticket.Resources)
		for _, id := range current {
			if strings.EqualFold(id, ar.Identifier) {
				out.Result, out.Reason = ResultSkipped, "already a resource"
				return out
			}
		}
		joined := strings.Join(append(current, ar.Identifier), ",")
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
