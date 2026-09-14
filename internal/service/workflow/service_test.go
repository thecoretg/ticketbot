package workflow

import (
	"context"
	"strings"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type fakeRecipRepo struct {
	repos.WebexRecipientRepository
	recips map[int]*models.WebexRecipient
}

func (f *fakeRecipRepo) Get(_ context.Context, id int) (*models.WebexRecipient, error) {
	r, ok := f.recips[id]
	if !ok {
		return nil, models.ErrWebexRecipientNotFound
	}
	return r, nil
}

func testService() *Service {
	return &Service{Recipients: &fakeRecipRepo{recips: map[int]*models.WebexRecipient{
		1: {ID: 1, Name: "Room", Type: models.RecipientTypeRoom},
		2: {ID: 2, Name: "Person", Type: models.RecipientTypePerson},
	}}}
}

func fields(errs models.ValidationErrors) []string {
	out := make([]string, 0, len(errs))
	for _, e := range errs {
		out = append(out, e.Field)
	}
	return out
}

func TestValidateAssignsIDsAndNormalizes(t *testing.T) {
	w := &models.Workflow{Rules: []models.Rule{
		{Name: "a", Enabled: true, Trigger: models.TriggerBoth},
	}}
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if w.Rules[0].ID == "" || len(w.Rules[0].ID) != 36 {
		t.Errorf("rule id not assigned: %q", w.Rules[0].ID)
	}
	if w.Rules[0].Actions == nil {
		t.Error("actions should be normalized to empty slice")
	}

	empty := &models.Workflow{}
	if errs := testService().Validate(context.Background(), empty); len(errs) != 0 || empty.Rules == nil {
		t.Errorf("empty workflow should validate and normalize rules: %v %v", errs, empty.Rules)
	}
}

func TestValidateValidWorkflow(t *testing.T) {
	w := &models.Workflow{Rules: []models.Rule{
		{Name: "a", Enabled: true, Trigger: models.TriggerCreate, Condition: "summary contains 'x' and company/id in (1,2)", Actions: []models.Action{
			notify(models.TargetRoom, rid(1)),
			notify(models.TargetPerson, rid(2)),
			notify(models.TargetResourcesOwner, nil),
			addNote("hello"),
			skip(),
		}},
	}}
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateErrors(t *testing.T) {
	bad := addNote("")
	bad.AddNote.Internal = false

	mismatch := notify(models.TargetRoom, rid(2)) // 2 is a person
	missing := notify(models.TargetRoom, rid(99))
	noRecip := notify(models.TargetPerson, nil)
	extra := notify(models.TargetResourcesOwner, rid(1))
	badTarget := notify(models.NotifyTarget("everyone"), nil)
	noSettings := models.Action{Kind: models.ActionNotify, Enabled: true}
	mixed := models.Action{Kind: models.ActionSkipNotify, Enabled: true, Notify: &models.NotifyAction{Target: models.TargetRoom}}
	unknown := models.Action{Kind: "teleport", Enabled: true}

	w := &models.Workflow{Rules: []models.Rule{
		{ID: "dup", Name: "", Trigger: "whenever", Condition: "summary =", Actions: []models.Action{
			bad, mismatch, missing, noRecip, extra, badTarget, noSettings, mixed, unknown,
		}},
		{ID: "dup", Name: "b", Trigger: models.TriggerBoth},
	}}

	errs := testService().Validate(context.Background(), w)
	got := strings.Join(fields(errs), ",")
	for _, want := range []string{
		"name", "trigger", "condition",
		"add_note.text", "add_note",
		"notify.recipient_id", "notify.target", "notify", "kind",
		"id",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}

	// condition error carries a position
	for _, e := range errs {
		if e.Field == "condition" && (e.Pos == nil || *e.Pos != 9) {
			t.Errorf("condition error pos = %v, want 9", e.Pos)
		}
		if e.Field == "add_note.text" && (e.ActionIndex == nil || *e.ActionIndex != 0) {
			t.Errorf("action index not set: %+v", e)
		}
	}

	if !strings.Contains(errs.Error(), "rule 1 action 2: notify.recipient_id") {
		t.Errorf("error text: %s", errs.Error())
	}
}

func TestValidateCondition(t *testing.T) {
	if ValidateCondition("") != nil || ValidateCondition("a = 1") != nil {
		t.Error("valid conditions should return nil")
	}
	se := ValidateCondition("a = ")
	if se == nil || se.Pos != 4 {
		t.Errorf("got %+v", se)
	}
}
