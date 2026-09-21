// Package workflow stores per-board workflows and walks their graphs against tickets.
package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/msgtemplate"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Workflows  repos.WorkflowRepository
	Recipients repos.WebexRecipientRepository
	Boards     repos.BoardRepository
	Statuses   repos.TicketStatusRepository
	Members    repos.MemberRepository
	// Lists is optional; when set, conditions referencing lists are checked on save.
	Lists repos.ListRepository
}

type Params struct {
	Workflows  repos.WorkflowRepository
	Recipients repos.WebexRecipientRepository
	Boards     repos.BoardRepository
	Statuses   repos.TicketStatusRepository
	Members    repos.MemberRepository
	Lists      repos.ListRepository
}

func New(p Params) *Service {
	return &Service{Workflows: p.Workflows, Recipients: p.Recipients, Boards: p.Boards, Statuses: p.Statuses, Members: p.Members, Lists: p.Lists}
}

func (s *Service) List(ctx context.Context) ([]*models.Workflow, error) {
	return s.Workflows.List(ctx)
}

func (s *Service) Get(ctx context.Context, id int) (*models.Workflow, error) {
	return s.Workflows.Get(ctx, id)
}

func (s *Service) GetByBoard(ctx context.Context, boardID int) (*models.Workflow, error) {
	return s.Workflows.GetByBoard(ctx, boardID)
}

// Create adds a workflow for a board that has none. An empty name defaults to the board name.
func (s *Service) Create(ctx context.Context, w *models.Workflow) (*models.Workflow, error) {
	board, err := s.Boards.Get(ctx, w.BoardID)
	if err != nil {
		return nil, fmt.Errorf("getting board %d: %w", w.BoardID, err)
	}

	exists, err := s.Workflows.ExistsForBoard(ctx, w.BoardID)
	if err != nil {
		return nil, fmt.Errorf("checking for existing workflow: %w", err)
	}
	if exists {
		return nil, models.ErrWorkflowExistsForBoard
	}

	if strings.TrimSpace(w.Name) == "" {
		w.Name = board.Name
	}
	w.BoardName = board.Name

	if errs := s.Validate(ctx, w); len(errs) > 0 {
		return nil, errs
	}

	return s.Workflows.Insert(ctx, w)
}

// Replace overwrites workflow id with w. The id and board come from the stored row, not the body.
func (s *Service) Replace(ctx context.Context, id int, w *models.Workflow) (*models.Workflow, error) {
	current, err := s.Workflows.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	w.ID = current.ID
	w.BoardID = current.BoardID
	w.BoardName = current.BoardName
	if strings.TrimSpace(w.Name) == "" {
		w.Name = current.Name
	}

	if errs := s.Validate(ctx, w); len(errs) > 0 {
		return nil, errs
	}

	return s.Workflows.Update(ctx, w)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if _, err := s.Workflows.Get(ctx, id); err != nil {
		return err
	}
	return s.Workflows.Delete(ctx, id)
}

// ValidateCondition compiles a condition string and returns its syntax error, if any.
func ValidateCondition(condition string) *cwquery.SyntaxError {
	_, se := parseCondition(condition)
	return se
}

func parseCondition(condition string) (cwquery.Expr, *cwquery.SyntaxError) {
	expr, err := cwquery.Parse(condition)
	if err == nil {
		return expr, nil
	}

	var se *cwquery.SyntaxError
	if errors.As(err, &se) {
		return nil, se
	}
	return nil, &cwquery.SyntaxError{Pos: 0, Msg: err.Error()}
}

// ListRefProblem is one bad `in list` reference: the list does not exist, or holds a different
// kind of item than the field being checked.
type ListRefProblem struct {
	Pos int
	Msg string
}

// ValidateListRefs checks every list reference in expr against stored lists. Fields the builder
// knows are also checked for type agreement (a contact field needs a contact list); unknown
// paths only need the list to exist.
func (s *Service) ValidateListRefs(ctx context.Context, expr cwquery.Expr) ([]ListRefProblem, error) {
	if s.Lists == nil {
		return nil, nil
	}

	var problems []ListRefProblem
	for _, ref := range cwquery.ListRefs(expr) {
		l, err := s.Lists.Get(ctx, ref.ListID)
		if err != nil {
			if errors.Is(err, models.ErrListNotFound) {
				problems = append(problems, ListRefProblem{Pos: ref.Pos, Msg: fmt.Sprintf("list %d not found", ref.ListID)})
				continue
			}
			return nil, fmt.Errorf("checking list %d: %w", ref.ListID, err)
		}

		f, ok := FieldByPath(strings.Join(ref.Path, "/"))
		if !ok || f.Source == "" {
			continue
		}
		want, ok := models.ListItemTypeForSource(f.Source)
		if !ok || want == l.ItemType {
			continue
		}
		have, _ := l.ItemType.Info()
		problems = append(problems, ListRefProblem{Pos: ref.Pos, Msg: fmt.Sprintf("list %q holds %s, but %s is a %s field", l.Name, strings.ToLower(have.Plural), f.Label, want)})
	}

	return problems, nil
}

// Validate checks the graph and every node's settings, assigns ids to new nodes and edges, and
// normalizes nil slices. It returns nil when the workflow is valid.
//
// Structural rules: at least one trigger; edges join existing nodes through a port the source
// node has, nothing wires into a trigger, one wire per port; no cycles; every node is reachable
// from a trigger. An if node may leave either port unwired: the path simply ends there.
func (s *Service) Validate(ctx context.Context, w *models.Workflow) models.ValidationErrors {
	var errs models.ValidationErrors
	nodeErr := func(id, field, msg string, pos *int) {
		errs = append(errs, models.ValidationError{NodeID: id, Field: field, Message: msg, Pos: pos})
	}
	edgeErr := func(id, field, msg string) {
		errs = append(errs, models.ValidationError{EdgeID: id, Field: field, Message: msg})
	}

	if w.Nodes == nil {
		w.Nodes = []models.Node{}
	}
	if w.Edges == nil {
		w.Edges = []models.Edge{}
	}

	nodes := make(map[string]*models.Node, len(w.Nodes))
	triggers := 0
	for i := range w.Nodes {
		n := &w.Nodes[i]
		if n.ID == "" {
			n.ID = models.NewID()
		}
		if _, dup := nodes[n.ID]; dup {
			nodeErr(n.ID, "id", "duplicate node id", nil)
		}
		nodes[n.ID] = n

		if strings.TrimSpace(n.Title) == "" {
			nodeErr(n.ID, "title", "title is required", nil)
		}
		if n.Kind == models.NodeTrigger {
			triggers++
		}
		for _, m := range s.validateNode(ctx, w.BoardID, n) {
			nodeErr(n.ID, m.field, m.text, m.pos)
		}
	}
	if triggers == 0 {
		errs = append(errs, models.ValidationError{Field: "nodes", Message: "a workflow needs at least one trigger"})
	}

	// edges: well-formed and one per port
	edges := make(map[string]bool, len(w.Edges))
	ports := make(map[string]string, len(w.Edges)) // "from/port" → edge id
	adj := make(map[string][]string, len(w.Nodes))
	for i := range w.Edges {
		ed := &w.Edges[i]
		if ed.ID == "" {
			ed.ID = models.NewID()
		}
		if edges[ed.ID] {
			edgeErr(ed.ID, "id", "duplicate edge id")
		}
		edges[ed.ID] = true

		from, okFrom := nodes[ed.From]
		to, okTo := nodes[ed.To]
		if !okFrom {
			edgeErr(ed.ID, "from", fmt.Sprintf("node %q does not exist", ed.From))
		}
		if !okTo {
			edgeErr(ed.ID, "to", fmt.Sprintf("node %q does not exist", ed.To))
		}
		if !okFrom || !okTo {
			continue
		}
		if ed.From == ed.To {
			edgeErr(ed.ID, "to", "a node cannot wire to itself")
			continue
		}
		if to.Kind == models.NodeTrigger {
			edgeErr(ed.ID, "to", "nothing can wire into a trigger")
			continue
		}
		if !from.HasPort(ed.Port) {
			edgeErr(ed.ID, "port", fmt.Sprintf("%s has no %q port", from.Kind, ed.Port))
			continue
		}
		key := ed.From + "/" + string(ed.Port)
		if other, taken := ports[key]; taken {
			edgeErr(ed.ID, "port", fmt.Sprintf("port already wired by edge %s", other))
			continue
		}
		ports[key] = ed.ID
		adj[ed.From] = append(adj[ed.From], ed.To)
	}

	if cyc := findCycle(nodes, adj); cyc != "" {
		nodeErr(cyc, "edges", "this node is part of a loop", nil)
	}

	// reachability: every non-trigger node needs a path from some trigger
	reached := make(map[string]bool, len(nodes))
	var visit func(id string)
	visit = func(id string) {
		if reached[id] {
			return
		}
		reached[id] = true
		for _, next := range adj[id] {
			visit(next)
		}
	}
	for _, n := range w.Nodes {
		if n.Kind == models.NodeTrigger {
			visit(n.ID)
		}
	}
	for _, n := range w.Nodes {
		if !reached[n.ID] {
			nodeErr(n.ID, "edges", "not connected to a trigger", nil)
		}
	}

	return errs
}

// findCycle returns the id of a node on a cycle, or "" when the graph is acyclic.
func findCycle(nodes map[string]*models.Node, adj map[string][]string) string {
	const (
		white = iota
		grey
		black
	)
	color := make(map[string]int, len(nodes))
	var dfs func(id string) string
	dfs = func(id string) string {
		color[id] = grey
		for _, next := range adj[id] {
			switch color[next] {
			case grey:
				return next
			case white:
				if c := dfs(next); c != "" {
					return c
				}
			}
		}
		color[id] = black
		return ""
	}
	ids := make([]string, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if color[id] == white {
			if c := dfs(id); c != "" {
				return c
			}
		}
	}
	return ""
}

type fieldMsg struct {
	field, text string
	pos         *int
}

// validateNode checks the fields a node's kind allows and requires.
func (s *Service) validateNode(ctx context.Context, boardID int, n *models.Node) []fieldMsg {
	var out []fieldMsg
	settings := hasSettings(n.ActionSettings)

	switch {
	case n.Kind == models.NodeTrigger:
		if len(n.Events) == 0 {
			out = append(out, fieldMsg{field: "events", text: "a trigger needs at least one event"})
		}
		seen := map[models.TriggerEvent]bool{}
		for _, ev := range n.Events {
			if !ev.Valid() {
				out = append(out, fieldMsg{field: "events", text: fmt.Sprintf("event must be created or updated (got %q)", ev)})
			}
			if seen[ev] {
				out = append(out, fieldMsg{field: "events", text: fmt.Sprintf("event %s listed twice", ev)})
			}
			seen[ev] = true
		}
		if n.Condition != "" {
			out = append(out, fieldMsg{field: "condition", text: "a trigger has no condition"})
		}
		if settings != "" {
			out = append(out, fieldMsg{field: settings, text: "a trigger has no action settings"})
		}

	case n.Kind == models.NodeIf:
		if len(n.Events) > 0 {
			out = append(out, fieldMsg{field: "events", text: "only a trigger has events"})
		}
		if settings != "" {
			out = append(out, fieldMsg{field: settings, text: "a condition has no action settings"})
		}
		if expr, se := parseCondition(n.Condition); se != nil {
			pos := se.Pos
			out = append(out, fieldMsg{field: "condition", text: se.Msg, pos: &pos})
		} else if problems, err := s.ValidateListRefs(ctx, expr); err != nil {
			out = append(out, fieldMsg{field: "condition", text: err.Error()})
		} else {
			for _, p := range problems {
				pos := p.Pos
				out = append(out, fieldMsg{field: "condition", text: p.Msg, pos: &pos})
			}
		}

	case n.Kind.IsAction():
		if len(n.Events) > 0 {
			out = append(out, fieldMsg{field: "events", text: "only a trigger has events"})
		}
		if n.Condition != "" {
			out = append(out, fieldMsg{field: "condition", text: "only an if node has a condition"})
		}
		a := n.Action()
		out = append(out, s.validateAction(ctx, boardID, &a)...)

	default:
		out = append(out, fieldMsg{field: "kind", text: fmt.Sprintf("unknown node kind %q", n.Kind)})
	}

	return out
}

// hasSettings names the first settings block that is set, or "".
func hasSettings(st models.ActionSettings) string {
	switch {
	case st.Notify != nil:
		return "notify"
	case st.AddNote != nil:
		return "add_note"
	case st.SetStatus != nil:
		return "set_status"
	case st.SetPriority != nil:
		return "set_priority"
	case st.SetOwner != nil:
		return "set_owner"
	case st.AddResource != nil:
		return "add_resource"
	case st.Patch != nil:
		return "patch"
	}
	return ""
}

// settingsFor maps each action kind to the JSON key of its settings block.
var settingsFor = map[models.ActionKind]string{
	models.ActionNotify:      "notify",
	models.ActionAddNote:     "add_note",
	models.ActionSetStatus:   "set_status",
	models.ActionSetPriority: "set_priority",
	models.ActionSetOwner:    "set_owner",
	models.ActionAddResource: "add_resource",
	models.ActionSkipNotify:  "",
}

func (s *Service) validateAction(ctx context.Context, boardID int, a *models.Action) []fieldMsg {
	if _, ok := settingsFor[a.Kind]; !ok && a.Kind != models.ActionPatch {
		return []fieldMsg{{field: "kind", text: fmt.Sprintf("unknown action kind %q", a.Kind)}}
	}

	// exactly the settings block for this kind may be present
	var out []fieldMsg
	present := []struct {
		key string
		set bool
	}{
		{"notify", a.Notify != nil},
		{"add_note", a.AddNote != nil},
		{"set_status", a.SetStatus != nil},
		{"set_priority", a.SetPriority != nil},
		{"set_owner", a.SetOwner != nil},
		{"add_resource", a.AddResource != nil},
		{"patch", a.Patch != nil},
	}
	want := settingsFor[a.Kind]
	if a.Kind == models.ActionPatch {
		want = "patch"
	}
	found := false
	for _, p := range present {
		switch {
		case !p.set:
		case p.key == want:
			found = true
		case a.Kind == models.ActionSkipNotify:
			out = append(out, fieldMsg{field: "kind", text: "skip_notify takes no settings"})
		default:
			out = append(out, fieldMsg{field: p.key, text: fmt.Sprintf("not allowed for a %s action", a.Kind)})
		}
	}
	if want != "" && !found {
		return append(out, fieldMsg{field: want, text: want + " settings are required"})
	}

	switch a.Kind {
	case models.ActionNotify:
		out = append(out, s.validateNotify(ctx, a.Notify)...)

	case models.ActionAddNote:
		if strings.TrimSpace(a.AddNote.Text) == "" {
			out = append(out, fieldMsg{field: "add_note.text", text: "note text is required"})
		}
		if !a.AddNote.Internal && !a.AddNote.Discussion && !a.AddNote.Resolution {
			out = append(out, fieldMsg{field: "add_note", text: "at least one of internal, discussion or resolution must be set"})
		}

	case models.ActionSetStatus:
		out = append(out, s.validateStatus(ctx, boardID, a.SetStatus)...)

	case models.ActionSetPriority:
		if a.SetPriority.PriorityID <= 0 {
			out = append(out, fieldMsg{field: "set_priority.priority_id", text: "priority is required"})
		}

	case models.ActionSetOwner:
		if msg, ok := s.lookupMember(ctx, a.SetOwner.MemberID, "set_owner.member_id", &a.SetOwner.Identifier); !ok {
			out = append(out, msg)
		}

	case models.ActionAddResource:
		if msg, ok := s.lookupMember(ctx, a.AddResource.MemberID, "add_resource.member_id", &a.AddResource.Identifier); !ok {
			out = append(out, msg)
		}

	case models.ActionPatch:
		ops, err := DecodePatchOps(a.Patch.Ops)
		if err != nil {
			out = append(out, fieldMsg{field: "patch.ops", text: err.Error()})
		} else if normalized, err := json.Marshal(ops); err == nil {
			a.Patch.Ops = normalized
		}
	}

	return out
}

func (s *Service) validateStatus(ctx context.Context, boardID int, st *models.SetStatusAction) []fieldMsg {
	if st.StatusID <= 0 {
		return []fieldMsg{{field: "set_status.status_id", text: "status is required"}}
	}
	rec, err := s.Statuses.Get(ctx, st.StatusID)
	if err != nil {
		return []fieldMsg{{field: "set_status.status_id", text: fmt.Sprintf("status %d not found", st.StatusID)}}
	}
	if rec.BoardID != boardID {
		return []fieldMsg{{field: "set_status.status_id", text: fmt.Sprintf("status %q belongs to another board", rec.Name)}}
	}
	if rec.Deleted || rec.Inactive {
		return []fieldMsg{{field: "set_status.status_id", text: fmt.Sprintf("status %q is inactive", rec.Name)}}
	}
	st.StatusName = rec.Name
	return nil
}

// lookupMember checks a member id and fills identifier from the member table.
func (s *Service) lookupMember(ctx context.Context, id int, field string, identifier *string) (fieldMsg, bool) {
	if id <= 0 {
		return fieldMsg{field: field, text: "member is required"}, false
	}
	m, err := s.Members.Get(ctx, id)
	if err != nil {
		return fieldMsg{field: field, text: fmt.Sprintf("member %d not found", id)}, false
	}
	if m.Deleted {
		return fieldMsg{field: field, text: fmt.Sprintf("member %s is deleted", m.Identifier)}, false
	}
	*identifier = m.Identifier
	return fieldMsg{}, true
}

func (s *Service) validateNotify(ctx context.Context, n *models.NotifyAction) []fieldMsg {
	var out []fieldMsg
	if err := msgtemplate.Validate(n.Message); err != nil {
		out = append(out, fieldMsg{field: "notify.message", text: err.Error()})
	}
	return append(out, s.validateNotifyTarget(ctx, n)...)
}

func (s *Service) validateNotifyTarget(ctx context.Context, n *models.NotifyAction) []fieldMsg {
	switch n.Target {
	case models.TargetResourcesOwner:
		if n.RecipientID != nil {
			return []fieldMsg{{field: "notify.recipient_id", text: "must be empty for resources_owner"}}
		}
		return nil

	case models.TargetRoom, models.TargetPerson:
		if n.RecipientID == nil {
			return []fieldMsg{{field: "notify.recipient_id", text: fmt.Sprintf("recipient is required for target %s", n.Target)}}
		}
		rec, err := s.Recipients.Get(ctx, *n.RecipientID)
		if err != nil {
			return []fieldMsg{{field: "notify.recipient_id", text: fmt.Sprintf("recipient %d not found", *n.RecipientID)}}
		}
		if string(rec.Type) != string(n.Target) {
			return []fieldMsg{{field: "notify.recipient_id", text: fmt.Sprintf("recipient %d is a %s, not a %s", rec.ID, rec.Type, n.Target)}}
		}
		return nil

	default:
		return []fieldMsg{{field: "notify.target", text: fmt.Sprintf("target must be one of room, person, resources_owner (got %q)", n.Target)}}
	}
}
