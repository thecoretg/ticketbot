// Package workflow stores per-board workflows and runs their rule chains against tickets.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
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

// Validate checks every rule and action, assigns IDs to new rules, and normalizes nil slices.
// It returns nil when the workflow is valid.
func (s *Service) Validate(ctx context.Context, w *models.Workflow) models.ValidationErrors {
	var errs models.ValidationErrors
	add := func(rule, action int, id, field, msg string, pos *int) {
		e := models.ValidationError{RuleIndex: rule, RuleID: id, Field: field, Message: msg, Pos: pos}
		if action >= 0 {
			a := action
			e.ActionIndex = &a
		}
		errs = append(errs, e)
	}

	if w.Rules == nil {
		w.Rules = []models.Rule{}
	}

	seen := make(map[string]bool, len(w.Rules))
	for i := range w.Rules {
		r := &w.Rules[i]

		if r.ID == "" {
			r.ID = newRuleID()
		}
		if seen[r.ID] {
			add(i, -1, r.ID, "id", "duplicate rule id", nil)
		}
		seen[r.ID] = true

		if strings.TrimSpace(r.Name) == "" {
			add(i, -1, r.ID, "name", "rule name is required", nil)
		}
		if !r.Trigger.Valid() {
			add(i, -1, r.ID, "trigger", fmt.Sprintf("trigger must be one of create, update, both (got %q)", r.Trigger), nil)
		}
		if expr, se := parseCondition(r.Condition); se != nil {
			pos := se.Pos
			add(i, -1, r.ID, "condition", se.Msg, &pos)
		} else if problems, err := s.ValidateListRefs(ctx, expr); err != nil {
			add(i, -1, r.ID, "condition", err.Error(), nil)
		} else {
			for _, p := range problems {
				pos := p.Pos
				add(i, -1, r.ID, "condition", p.Msg, &pos)
			}
		}

		if r.Actions == nil {
			r.Actions = []models.Action{}
		}
		for j := range r.Actions {
			for _, msg := range s.validateAction(ctx, w.BoardID, &r.Actions[j]) {
				add(i, j, r.ID, msg.field, msg.text, nil)
			}
		}
	}

	return errs
}

type fieldMsg struct{ field, text string }

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
		return []fieldMsg{{"kind", fmt.Sprintf("unknown action kind %q", a.Kind)}}
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
			out = append(out, fieldMsg{"kind", "skip_notify takes no settings"})
		default:
			out = append(out, fieldMsg{p.key, fmt.Sprintf("not allowed for a %s action", a.Kind)})
		}
	}
	if want != "" && !found {
		return append(out, fieldMsg{want, want + " settings are required"})
	}

	switch a.Kind {
	case models.ActionNotify:
		out = append(out, s.validateNotify(ctx, a.Notify)...)

	case models.ActionAddNote:
		if strings.TrimSpace(a.AddNote.Text) == "" {
			out = append(out, fieldMsg{"add_note.text", "note text is required"})
		}
		if !a.AddNote.Internal && !a.AddNote.Discussion && !a.AddNote.Resolution {
			out = append(out, fieldMsg{"add_note", "at least one of internal, discussion or resolution must be set"})
		}

	case models.ActionSetStatus:
		out = append(out, s.validateStatus(ctx, boardID, a.SetStatus)...)

	case models.ActionSetPriority:
		if a.SetPriority.PriorityID <= 0 {
			out = append(out, fieldMsg{"set_priority.priority_id", "priority is required"})
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
			out = append(out, fieldMsg{"patch.ops", err.Error()})
		} else if normalized, err := json.Marshal(ops); err == nil {
			a.Patch.Ops = normalized
		}
	}

	return out
}

func (s *Service) validateStatus(ctx context.Context, boardID int, st *models.SetStatusAction) []fieldMsg {
	if st.StatusID <= 0 {
		return []fieldMsg{{"set_status.status_id", "status is required"}}
	}
	rec, err := s.Statuses.Get(ctx, st.StatusID)
	if err != nil {
		return []fieldMsg{{"set_status.status_id", fmt.Sprintf("status %d not found", st.StatusID)}}
	}
	if rec.BoardID != boardID {
		return []fieldMsg{{"set_status.status_id", fmt.Sprintf("status %q belongs to another board", rec.Name)}}
	}
	if rec.Deleted || rec.Inactive {
		return []fieldMsg{{"set_status.status_id", fmt.Sprintf("status %q is inactive", rec.Name)}}
	}
	st.StatusName = rec.Name
	return nil
}

// lookupMember checks a member id and fills identifier from the member table.
func (s *Service) lookupMember(ctx context.Context, id int, field string, identifier *string) (fieldMsg, bool) {
	if id <= 0 {
		return fieldMsg{field, "member is required"}, false
	}
	m, err := s.Members.Get(ctx, id)
	if err != nil {
		return fieldMsg{field, fmt.Sprintf("member %d not found", id)}, false
	}
	if m.Deleted {
		return fieldMsg{field, fmt.Sprintf("member %s is deleted", m.Identifier)}, false
	}
	*identifier = m.Identifier
	return fieldMsg{}, true
}

func (s *Service) validateNotify(ctx context.Context, n *models.NotifyAction) []fieldMsg {
	var out []fieldMsg
	if err := msgtemplate.Validate(n.Message); err != nil {
		out = append(out, fieldMsg{"notify.message", err.Error()})
	}
	return append(out, s.validateNotifyTarget(ctx, n)...)
}

func (s *Service) validateNotifyTarget(ctx context.Context, n *models.NotifyAction) []fieldMsg {
	switch n.Target {
	case models.TargetResourcesOwner:
		if n.RecipientID != nil {
			return []fieldMsg{{"notify.recipient_id", "must be empty for resources_owner"}}
		}
		return nil

	case models.TargetRoom, models.TargetPerson:
		if n.RecipientID == nil {
			return []fieldMsg{{"notify.recipient_id", fmt.Sprintf("recipient is required for target %s", n.Target)}}
		}
		rec, err := s.Recipients.Get(ctx, *n.RecipientID)
		if err != nil {
			return []fieldMsg{{"notify.recipient_id", fmt.Sprintf("recipient %d not found", *n.RecipientID)}}
		}
		if string(rec.Type) != string(n.Target) {
			return []fieldMsg{{"notify.recipient_id", fmt.Sprintf("recipient %d is a %s, not a %s", rec.ID, rec.Type, n.Target)}}
		}
		return nil

	default:
		return []fieldMsg{{"notify.target", fmt.Sprintf("target must be one of room, person, resources_owner (got %q)", n.Target)}}
	}
}

func newRuleID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand unavailable: %v", err))
	}
	// RFC 4122 version 4 layout
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b[:])
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:]
}
