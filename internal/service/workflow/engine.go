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

// NotifyIntent is a notify node that ran; the caller resolves the channel's recipients and sends.
type NotifyIntent struct {
	Step   StepRef
	Action models.NotifyAction
}

// Action results.
const (
	ResultOK       = "ok"
	ResultWouldRun = "would_run"
	ResultQueued   = "queued"
	ResultSkipped  = "skipped"
	ResultError    = "error"
	// ResultSuperseded marks a queued ticket write a later node in the same run replaced: the
	// batched PATCH carries one operation per path, and the last node to set it wins.
	ResultSuperseded = "superseded"
)

// WriteLimiter caps ConnectWise writes per ticket. Reserve records n writes for the ticket, or
// returns an error naming when the ticket is unblocked. A nil limiter allows everything.
type WriteLimiter interface {
	Reserve(ticketID int, n int) error
}

// Conflict records two nodes in one run setting the same ticket field.
type Conflict struct {
	Path       string
	Superseded StepRef
	By         StepRef
}

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
	Matched *bool       // if nodes, and triggers with a condition: the condition's verdict
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
	// Conflicts lists fields two nodes set; the later node's value was written.
	Conflicts []Conflict
	CWWrites  int
	// RateCapped is set when the write cap refused this run's ConnectWise writes.
	RateCapped bool
	// LearnedAPIMember is the member identifier ConnectWise attributed to a write the engine made.
	LearnedAPIMember string
}

type Engine struct {
	CW CWClient
	// Lists is optional; set it to make `in list` conditions see admin lists.
	Lists ListLoader
	// Limiter is optional; set it to cap writes per ticket.
	Limiter WriteLimiter
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
	// Ticket writes are collected during the walks and sent afterwards: one PATCH carrying every
	// field operation, then each note. Later nodes still see the intended state because every
	// queued operation is applied to the in-memory ticket as it is queued.
	ops   []pendingOp
	notes []pendingNote
	dirty bool // the in-memory ticket changed since the document was last built
}

// pendingOp is one field operation waiting for the batched PATCH. actions index r.res.Actions:
// usually one, more when add_resource nodes merged into a single resources value.
type pendingOp struct {
	op      psa.PatchOp
	actions []int
	step    StepRef
}

type pendingNote struct {
	note   *models.AddNoteAction
	action int
}

// Run walks the workflow graph for the ticket. Every enabled trigger listening for the intake's
// event starts a walk, in canvas order; each walk follows the port its nodes select until a port
// has no wire, a node another walk already ran is reached, or the step cap trips. Notify nodes are
// collected as intents for the caller. Ticket-mutating nodes are queued: field operations merge
// into one PATCH (the last node to set a path wins and the earlier one is marked superseded) and
// notes follow it, all sent after the walks unless DryRun. Later nodes see the queued state because
// each operation is applied to the in-memory ticket as it is queued. Errors inside a node are
// recorded, never returned; only a nil workflow or ticket is an error.
func (e *Engine) Run(ctx context.Context, wf *models.Workflow, in Input) (*Result, error) {
	if wf == nil {
		return nil, errors.New("nil workflow")
	}
	if in.Ticket == nil {
		return nil, errors.New("nil ticket")
	}

	// Queued writes are applied to a copy so the caller's ticket stays what ConnectWise holds.
	local := *in.Ticket
	r := &run{
		wf: wf,
		in: in,
		res: &Result{
			Ticket:      &local,
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

	e.flush(ctx, r)

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
			// A trigger's own condition gates the lane: when it fails or errors the walk never
			// starts, and the step is still recorded so the run shows the lane did not fire.
			if strings.TrimSpace(cur.Condition) != "" {
				matched, err := e.evalCondition(ctx, r, cur.Condition)
				if err != nil {
					st.Err = err
					r.res.Steps = append(r.res.Steps, st)
					return
				}
				st.Matched = &matched
				if !matched {
					r.res.Steps = append(r.res.Steps, st)
					return
				}
			}

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
			out := e.runAction(ctx, ref, st.No, cur.Action(), r, &suppressed)
			r.res.Actions = append(r.res.Actions, out)
			if r.dirty {
				r.doc = cwquery.NewDocument(r.res.Ticket, r.in.TriggerNote, r.changes)
				r.dirty = false
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
	res := r.res
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
		a.Notify.Normalize()
		res.Notifies = append(res.Notifies, NotifyIntent{Step: ref, Action: *a.Notify})
		out.Result = ResultQueued
		out.Output = map[string]any{"channel": string(a.Notify.Channel)}
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
		out.Output = map[string]any{
			"text":       a.AddNote.Text,
			"internal":   a.AddNote.Internal,
			"discussion": a.AddNote.Discussion,
			"resolution": a.AddNote.Resolution,
		}
		r.notes = append(r.notes, pendingNote{note: a.AddNote, action: len(res.Actions)})
		out.Result = ResultQueued

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
		res.Ticket.Status.ID, res.Ticket.Status.Name = s.StatusID, s.StatusName
		r.queue(ref, psa.PatchOp{Op: "replace", Path: "status/id", Value: s.StatusID}, &out)

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
		res.Ticket.Priority.ID, res.Ticket.Priority.Name = p.PriorityID, p.PriorityName
		r.queue(ref, psa.PatchOp{Op: "replace", Path: "priority/id", Value: p.PriorityID}, &out)

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
		res.Ticket.Owner.ID, res.Ticket.Owner.Identifier = o.MemberID, o.Identifier
		r.queue(ref, psa.PatchOp{Op: "replace", Path: "owner/id", Value: o.MemberID}, &out)

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
		res.Ticket.Resources = joined
		r.queue(ref, psa.PatchOp{Op: "replace", Path: "resources", Value: joined}, &out)

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
		for _, op := range ops {
			r.queue(ref, op, &out)
		}

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

// queue adds one field operation to the run's PATCH. A second operation on the same path
// replaces the first: the earlier action is marked superseded and the pair is recorded as a
// conflict, unless both are add_resource, whose later value already includes the earlier member.
func (r *run) queue(ref StepRef, op psa.PatchOp, out *ActionOutcome) {
	out.Result = ResultQueued
	r.dirty = true
	idx := len(r.res.Actions)

	for i := range r.ops {
		prev := &r.ops[i]
		if prev.op.Path != op.Path {
			continue
		}
		merge := op.Path == "resources" && out.Kind == models.ActionAddResource && r.res.Actions[prev.actions[0]].Kind == models.ActionAddResource
		if !merge {
			r.res.Conflicts = append(r.res.Conflicts, Conflict{Path: op.Path, Superseded: prev.step, By: ref})
			for _, ai := range prev.actions {
				a := &r.res.Actions[ai]
				a.Result, a.Reason = ResultSuperseded, "overridden by "+ref.Title
			}
			slog.Warn("workflow: two nodes set the same field; the later one wins",
				"workflow_id", r.wf.ID, "path", op.Path, "superseded", prev.step.Title, "by", ref.Title)
			prev.actions = nil
		}
		prev.op, prev.step = op, ref
		prev.actions = append(prev.actions, idx)
		return
	}

	r.ops = append(r.ops, pendingOp{op: op, actions: []int{idx}, step: ref})
}

// flush sends the queued writes: the PATCH first, then each note, then one refetch so the caller
// stores what ConnectWise now holds. In a dry run every queued action reports would_run instead.
func (e *Engine) flush(ctx context.Context, r *run) {
	res := r.res
	if len(r.ops) == 0 && len(r.notes) == 0 {
		return
	}

	if r.in.DryRun {
		for _, p := range r.ops {
			for _, ai := range p.actions {
				res.Actions[ai].Result = ResultWouldRun
			}
		}
		for _, n := range r.notes {
			res.Actions[n.action].Result = ResultWouldRun
		}
		res.Ticket = r.in.Ticket // nothing was written; hand back the real state
		return
	}

	needRefetch := false
	if len(r.ops) > 0 {
		ops := make([]psa.PatchOp, 0, len(r.ops))
		for _, p := range r.ops {
			ops = append(ops, p.op)
		}
		err := e.reserve(res.Ticket.ID, 1)
		if err == nil {
			var t *psa.Ticket
			t, err = e.CW.PatchTicket(ctx, res.Ticket.ID, ops)
			if err != nil {
				err = fmt.Errorf("patching ticket: %w", err)
			} else {
				res.CWWrites++
				if t != nil && t.ID != 0 {
					res.Ticket = t
					if t.Info.UpdatedBy != "" {
						res.LearnedAPIMember = t.Info.UpdatedBy
					}
				} else {
					needRefetch = true
				}
			}
		} else {
			res.RateCapped = true
		}
		if err != nil {
			res.Ticket = r.in.Ticket // the queued state was never written
		}
		for _, p := range r.ops {
			for _, ai := range p.actions {
				a := &res.Actions[ai]
				if err != nil {
					a.Result, a.Err = ResultError, err
				} else {
					a.Result = ResultOK
				}
			}
		}
	}

	for _, pn := range r.notes {
		a := &res.Actions[pn.action]
		if err := e.reserve(res.Ticket.ID, 1); err != nil {
			res.RateCapped = true
			a.Result, a.Err = ResultError, err
			continue
		}
		n := pn.note
		posted, err := e.CW.PostServiceTicketNote(ctx, &psa.ServiceTicketNote{
			Text:                  n.Text,
			InternalAnalysisFlag:  n.Internal,
			DetailDescriptionFlag: n.Discussion,
			ResolutionFlag:        n.Resolution,
		}, res.Ticket.ID)
		if err != nil {
			a.Result, a.Err = ResultError, fmt.Errorf("posting note: %w", err)
			continue
		}
		res.CWWrites++
		needRefetch = true
		a.Result = ResultOK
		if posted != nil {
			a.Output["note_id"] = posted.ID
			res.LearnedAPIMember = posted.Member.Identifier
			res.LatestNote = posted
		}
	}

	if needRefetch {
		if err := e.refetch(ctx, res); err != nil {
			// the writes succeeded; a stale local copy is a warning, not an action failure
			slog.Warn("workflow: refetching ticket after writes", "ticket_id", res.Ticket.ID, "error", err.Error())
		}
	}
}

// reserve asks the limiter for n writes; without a limiter every write is allowed.
func (e *Engine) reserve(ticketID, n int) error {
	if e.Limiter == nil {
		return nil
	}
	return e.Limiter.Reserve(ticketID, n)
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
