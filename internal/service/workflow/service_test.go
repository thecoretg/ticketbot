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

type fakeStatusRepo struct {
	repos.TicketStatusRepository
	statuses map[int]*models.TicketStatus
}

func (f *fakeStatusRepo) Get(_ context.Context, id int) (*models.TicketStatus, error) {
	s, ok := f.statuses[id]
	if !ok {
		return nil, models.ErrTicketStatusNotFound
	}
	return s, nil
}

type fakeMemberRepo struct {
	repos.MemberRepository
	members map[int]*models.Member
}

func (f *fakeMemberRepo) Get(_ context.Context, id int) (*models.Member, error) {
	m, ok := f.members[id]
	if !ok {
		return nil, models.ErrMemberNotFound
	}
	return m, nil
}

type fakeListRepo struct {
	repos.ListRepository
	lists map[int]*models.List
}

func (f *fakeListRepo) Get(_ context.Context, id int) (*models.List, error) {
	l, ok := f.lists[id]
	if !ok {
		return nil, models.ErrListNotFound
	}
	return l, nil
}

func testService() *Service {
	return &Service{
		Lists: &fakeListRepo{lists: map[int]*models.List{
			7: {ID: 7, Name: "Drop", ItemType: models.ListItemContact},
			8: {ID: 8, Name: "VIPs", ItemType: models.ListItemCompany},
		}},
		Recipients: &fakeRecipRepo{recips: map[int]*models.WebexRecipient{
			1: {ID: 1, Name: "Room", Type: models.RecipientTypeRoom},
			2: {ID: 2, Name: "Person", Type: models.RecipientTypePerson},
		}},
		Statuses: &fakeStatusRepo{statuses: map[int]*models.TicketStatus{
			10: {ID: 10, BoardID: 1, Name: "Escalated"},
			20: {ID: 20, BoardID: 2, Name: "Other Board"},
			30: {ID: 30, BoardID: 1, Name: "Retired", Inactive: true},
		}},
		Members: &fakeMemberRepo{members: map[int]*models.Member{
			5: {ID: 5, Identifier: "jdoe"},
			6: {ID: 6, Identifier: "gone", Deleted: true},
		}},
	}
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

func TestValidateMutatingActions(t *testing.T) {
	w := &models.Workflow{BoardID: 1, Rules: []models.Rule{
		{Name: "a", Enabled: true, Trigger: models.TriggerBoth, Actions: []models.Action{
			{Kind: models.ActionSetStatus, Enabled: true, SetStatus: &models.SetStatusAction{StatusID: 10}},
			{Kind: models.ActionSetPriority, Enabled: true, SetPriority: &models.SetPriorityAction{PriorityID: 3, PriorityName: "P3"}},
			{Kind: models.ActionSetOwner, Enabled: true, SetOwner: &models.SetOwnerAction{MemberID: 5}},
			{Kind: models.ActionAddResource, Enabled: true, AddResource: &models.AddResourceAction{MemberID: 5, Identifier: "stale"}},
			{Kind: models.ActionPatch, Enabled: true, Patch: &models.PatchAction{Ops: []byte(` [ {"op":"replace", "path":"severity", "value":"High"} ] `)}},
			{Kind: models.ActionNotify, Enabled: true, Notify: &models.NotifyAction{Target: models.TargetRoom, RecipientID: rid(1), Message: "{{event}} {{ticket.link}} for {{company}}"}},
		}},
	}}
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	acts := w.Rules[0].Actions
	if acts[0].SetStatus.StatusName != "Escalated" {
		t.Errorf("status name not filled: %+v", acts[0].SetStatus)
	}
	if acts[2].SetOwner.Identifier != "jdoe" || acts[3].AddResource.Identifier != "jdoe" {
		t.Errorf("member identifiers not filled: %+v %+v", acts[2].SetOwner, acts[3].AddResource)
	}
	if string(acts[4].Patch.Ops) != `[{"op":"replace","path":"severity","value":"High"}]` {
		t.Errorf("patch ops not normalized: %s", acts[4].Patch.Ops)
	}
}

func TestValidateMutatingActionErrors(t *testing.T) {
	w := &models.Workflow{BoardID: 1, Rules: []models.Rule{
		{Name: "a", Enabled: true, Trigger: models.TriggerBoth, Actions: []models.Action{
			{Kind: models.ActionSetStatus, Enabled: true, SetStatus: &models.SetStatusAction{StatusID: 20}},                                            // other board
			{Kind: models.ActionSetStatus, Enabled: true, SetStatus: &models.SetStatusAction{StatusID: 30}},                                            // inactive
			{Kind: models.ActionSetStatus, Enabled: true, SetStatus: &models.SetStatusAction{StatusID: 99}},                                            // missing
			{Kind: models.ActionSetStatus, Enabled: true},                                                                                              // no settings
			{Kind: models.ActionSetPriority, Enabled: true, SetPriority: &models.SetPriorityAction{}},                                                  // no id
			{Kind: models.ActionSetOwner, Enabled: true, SetOwner: &models.SetOwnerAction{MemberID: 6}},                                                // deleted
			{Kind: models.ActionAddResource, Enabled: true, AddResource: &models.AddResourceAction{MemberID: 99}},                                      // missing
			{Kind: models.ActionPatch, Enabled: true, Patch: &models.PatchAction{Ops: []byte(`{"op":"replace"}`)}},                                     // not an array
			{Kind: models.ActionPatch, Enabled: true, Patch: &models.PatchAction{Ops: []byte(`[]`)}, SetStatus: &models.SetStatusAction{StatusID: 10}}, // mixed
			{Kind: models.ActionNotify, Enabled: true, Notify: &models.NotifyAction{Target: models.TargetResourcesOwner, Message: "hi {{nope}}"}},
		}},
	}}
	errs := testService().Validate(context.Background(), w)
	want := []string{
		"set_status.status_id", "set_status.status_id", "set_status.status_id", "set_status",
		"set_priority.priority_id", "set_owner.member_id", "add_resource.member_id",
		"patch.ops", "set_status", "patch.ops", "notify.message",
	}
	got := fields(errs)
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("fields = %v\nwant     %v\nerrs: %v", got, want, errs)
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

func TestValidateListRefs(t *testing.T) {
	cases := []struct {
		cond string
		want string // substring of the message; empty means valid
	}{
		{"contact/id in list 7", ""},
		{"latestNote/contact/id not in list 7", ""},
		{"company/id in list 8", ""},
		{"summary in list 7", ""}, // unknown type: existence only
		{"contact/id in list 99", "list 99 not found"},
		{"company/id in list 7", `list "Drop" holds contacts, but Company is a company field`},
		{"contact/id in list 8", `list "VIPs" holds companies, but Contact is a contact field`},
	}
	for _, c := range cases {
		w := &models.Workflow{Rules: []models.Rule{{Name: "r", Enabled: true, Trigger: models.TriggerBoth, Condition: c.cond}}}
		errs := testService().Validate(context.Background(), w)
		if c.want == "" {
			if len(errs) != 0 {
				t.Errorf("%q: unexpected errors %v", c.cond, errs)
			}
			continue
		}
		if len(errs) != 1 || errs[0].Field != "condition" || !strings.Contains(errs[0].Message, c.want) || errs[0].Pos == nil {
			t.Errorf("%q: errs = %v, want message containing %q with pos", c.cond, errs, c.want)
			continue
		}
		if *errs[0].Pos != strings.LastIndex(c.cond, " ")+1 {
			t.Errorf("%q: pos = %d", c.cond, *errs[0].Pos)
		}
	}

	// without a list repo the check is skipped
	svc := testService()
	svc.Lists = nil
	w := &models.Workflow{Rules: []models.Rule{{Name: "r", Enabled: true, Trigger: models.TriggerBoth, Condition: "contact/id in list 99"}}}
	if errs := svc.Validate(context.Background(), w); len(errs) != 0 {
		t.Errorf("nil Lists: unexpected errors %v", errs)
	}
}
