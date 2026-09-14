// Package workflow stores per-board workflows and runs their rule chains against tickets.
package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Workflows  repos.WorkflowRepository
	Recipients repos.WebexRecipientRepository
	Boards     repos.BoardRepository
}

func New(w repos.WorkflowRepository, r repos.WebexRecipientRepository, b repos.BoardRepository) *Service {
	return &Service{Workflows: w, Recipients: r, Boards: b}
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
	_, err := cwquery.Compile(condition)
	if err == nil {
		return nil
	}

	var se *cwquery.SyntaxError
	if errors.As(err, &se) {
		return se
	}
	return &cwquery.SyntaxError{Pos: 0, Msg: err.Error()}
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
		if se := ValidateCondition(r.Condition); se != nil {
			pos := se.Pos
			add(i, -1, r.ID, "condition", se.Msg, &pos)
		}

		if r.Actions == nil {
			r.Actions = []models.Action{}
		}
		for j := range r.Actions {
			for _, msg := range s.validateAction(ctx, &r.Actions[j]) {
				add(i, j, r.ID, msg.field, msg.text, nil)
			}
		}
	}

	return errs
}

type fieldMsg struct{ field, text string }

func (s *Service) validateAction(ctx context.Context, a *models.Action) []fieldMsg {
	var out []fieldMsg

	switch a.Kind {
	case models.ActionNotify:
		if a.AddNote != nil {
			out = append(out, fieldMsg{"add_note", "not allowed for a notify action"})
		}
		if a.Notify == nil {
			return append(out, fieldMsg{"notify", "notify settings are required"})
		}
		out = append(out, s.validateNotify(ctx, a.Notify)...)

	case models.ActionAddNote:
		if a.Notify != nil {
			out = append(out, fieldMsg{"notify", "not allowed for an add_note action"})
		}
		if a.AddNote == nil {
			return append(out, fieldMsg{"add_note", "note settings are required"})
		}
		if strings.TrimSpace(a.AddNote.Text) == "" {
			out = append(out, fieldMsg{"add_note.text", "note text is required"})
		}
		if !a.AddNote.Internal && !a.AddNote.Discussion && !a.AddNote.Resolution {
			out = append(out, fieldMsg{"add_note", "at least one of internal, discussion or resolution must be set"})
		}

	case models.ActionSkipNotify:
		if a.Notify != nil || a.AddNote != nil {
			out = append(out, fieldMsg{"kind", "skip_notify takes no settings"})
		}

	default:
		out = append(out, fieldMsg{"kind", fmt.Sprintf("unknown action kind %q", a.Kind)})
	}

	return out
}

func (s *Service) validateNotify(ctx context.Context, n *models.NotifyAction) []fieldMsg {
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
