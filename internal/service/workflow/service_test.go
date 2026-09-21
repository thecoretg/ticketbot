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

// flow wires a trigger to the nodes in a straight line, leaving an if by its yes port.
func flow(nodes ...models.Node) *models.Workflow {
	all := append([]models.Node{trig("t", 0)}, nodes...)
	var edges []models.Edge
	prev := all[0]
	for _, n := range nodes {
		edges = append(edges, edge(prev.ID, prev.Ports()[0], n.ID))
		prev = n
	}
	return &models.Workflow{BoardID: 1, Nodes: all, Edges: edges}
}

func TestValidateAssignsIDsAndNormalizes(t *testing.T) {
	w := &models.Workflow{Nodes: []models.Node{
		{Kind: models.NodeTrigger, Title: "t", Enabled: true, Events: []models.TriggerEvent{models.TriggerCreated}},
		{Kind: models.NodeIf, Title: "c", Enabled: true},
	}}
	w.Edges = nil
	if errs := testService().Validate(context.Background(), w); len(errs) != 1 || errs[0].Message != "not connected to a trigger" {
		t.Fatalf("errors = %v", errs)
	}
	if len(w.Nodes[0].ID) != 36 || len(w.Nodes[1].ID) != 36 {
		t.Errorf("node ids not assigned: %+v", w.Nodes)
	}
	if w.Edges == nil {
		t.Error("edges should be normalized to an empty slice")
	}

	w.Edges = []models.Edge{{From: w.Nodes[0].ID, To: w.Nodes[1].ID, Port: models.PortOut}}
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if len(w.Edges[0].ID) != 36 {
		t.Errorf("edge id not assigned: %+v", w.Edges[0])
	}

	empty := &models.Workflow{}
	errs := testService().Validate(context.Background(), empty)
	if len(errs) != 1 || errs[0].Field != "nodes" || empty.Nodes == nil || empty.Edges == nil {
		t.Errorf("empty workflow: errs=%v nodes=%v edges=%v", errs, empty.Nodes, empty.Edges)
	}
}

func TestValidateValidWorkflow(t *testing.T) {
	w := flow(
		iff("c", "summary contains 'x' and company/id in (1,2)"),
		act("n1", notify(models.TargetRoom, rid(1))),
		act("n2", notify(models.TargetPerson, rid(2))),
		act("n3", notify(models.TargetResourcesOwner, nil)),
		act("a", addNote("hello")),
		act("s", skip()),
	)
	// the else branch may end unwired, and a node may take several inputs
	w.Nodes = append(w.Nodes, act("end", notify(models.TargetRoom, rid(1))))
	w.Edges = append(w.Edges, edge("s", models.PortOut, "end"), edge("c", models.PortNo, "end"))
	// wait: s already leaves by out; move that edge onto the join instead
	w.Edges = w.Edges[:len(w.Edges)-2]
	w.Edges = append(w.Edges, edge("s", models.PortOut, "end"), edge("c", models.PortNo, "end"))
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}

func TestValidateStructure(t *testing.T) {
	cases := []struct {
		name string
		w    *models.Workflow
		want []string // one "field:message-substring" per expected error, in order
	}{
		{"no trigger", &models.Workflow{Nodes: []models.Node{iff("c", "")}}, []string{"nodes:at least one trigger", "edges:not connected"}},
		{"dup ids", func() *models.Workflow {
			w := flow(iff("c", ""))
			w.Nodes = append(w.Nodes, iff("c", ""))
			return w
		}(), []string{"id:duplicate node id"}},
		{"trigger needs events", &models.Workflow{Nodes: []models.Node{{ID: "t", Kind: models.NodeTrigger, Title: "t", Enabled: true}}}, []string{"events:at least one event"}},
		{"bad and duplicate events", &models.Workflow{Nodes: []models.Node{{ID: "t", Kind: models.NodeTrigger, Title: "t", Events: []models.TriggerEvent{"deleted", models.TriggerCreated, models.TriggerCreated}}}}, []string{"events:created or updated", "events:listed twice"}},
		{"trigger with condition and settings", &models.Workflow{Nodes: []models.Node{{ID: "t", Kind: models.NodeTrigger, Title: "t", Events: []models.TriggerEvent{models.TriggerCreated}, Condition: "id = 1", ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{}}}}}, []string{"condition:no condition", "notify:no action settings"}},
		{"if with events and settings", flow(models.Node{ID: "c", Kind: models.NodeIf, Title: "c", Events: []models.TriggerEvent{models.TriggerCreated}, ActionSettings: models.ActionSettings{AddNote: &models.AddNoteAction{}}}), []string{"events:only a trigger", "add_note:no action settings"}},
		{"action with condition", flow(models.Node{ID: "n", Kind: "skip_notify", Title: "n", Condition: "id = 1"}), []string{"condition:only an if node"}},
		{"unknown kind", flow(models.Node{ID: "x", Kind: "teleport", Title: "x"}), []string{"kind:unknown node kind"}},
		{"untitled", flow(models.Node{ID: "c", Kind: models.NodeIf, Enabled: true}), []string{"title:title is required"}},
		{"edge to missing node", func() *models.Workflow {
			w := flow(iff("c", ""))
			w.Edges = append(w.Edges, edge("c", models.PortYes, "ghost"), edge("ghost", models.PortOut, "c"))
			return w
		}(), []string{"to:does not exist", "from:does not exist"}},
		{"self loop", func() *models.Workflow {
			w := flow(act("n", skip()))
			w.Edges = append(w.Edges, edge("n", models.PortOut, "n"))
			return w
		}(), []string{"to:cannot wire to itself"}},
		{"into a trigger", func() *models.Workflow {
			w := flow(act("n", skip()))
			w.Edges = append(w.Edges, edge("n", models.PortOut, "t"))
			return w
		}(), []string{"to:into a trigger"}},
		{"wrong port", func() *models.Workflow {
			w := flow(act("n", skip()), act("m", skip()))
			w.Edges[1].Port = models.PortYes
			return w
		}(), []string{"port:has no \"yes\" port", "edges:not connected"}},
		{"port wired twice", func() *models.Workflow {
			w := flow(act("n", skip()), act("m", skip()))
			w.Edges = append(w.Edges, edge("n", models.PortOut, "m"))
			w.Edges[2].ID = "other"
			return w
		}(), []string{"port:already wired"}},
		{"cycle", func() *models.Workflow {
			w := flow(act("a", skip()), act("b", skip()), act("c", skip()))
			w.Edges = append(w.Edges, edge("c", models.PortOut, "b"))
			return w
		}(), []string{"edges:part of a loop"}},
		{"orphan", func() *models.Workflow {
			w := flow(act("a", skip()))
			w.Nodes = append(w.Nodes, act("lonely", skip()))
			return w
		}(), []string{"edges:not connected"}},
	}
	for _, c := range cases {
		errs := testService().Validate(context.Background(), c.w)
		var got []string
		for _, e := range errs {
			got = append(got, e.Field+":"+e.Message)
		}
		if len(got) != len(c.want) {
			t.Errorf("%s: errors = %v, want %v", c.name, got, c.want)
			continue
		}
		for i := range c.want {
			f, msg, _ := strings.Cut(c.want[i], ":")
			if !strings.HasPrefix(got[i], f+":") || !strings.Contains(got[i], msg) {
				t.Errorf("%s: error %d = %q, want %q", c.name, i, got[i], c.want[i])
			}
		}
	}
}

func TestValidateErrorsCarryNodeAndPosition(t *testing.T) {
	bad := addNote("")
	bad.AddNote.Internal = false

	mismatch := notify(models.TargetRoom, rid(2)) // 2 is a person
	missing := notify(models.TargetRoom, rid(99))
	noRecip := notify(models.TargetPerson, nil)
	extra := notify(models.TargetResourcesOwner, rid(1))
	badTarget := notify(models.NotifyTarget("everyone"), nil)
	noSettings := models.Action{Kind: models.ActionNotify, Enabled: true}
	mixed := models.Action{Kind: models.ActionSkipNotify, Enabled: true, ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Target: models.TargetRoom}}}

	w := flow(iff("cond", "summary ="),
		act("bad", bad), act("mismatch", mismatch), act("missing", missing), act("norecip", noRecip),
		act("extra", extra), act("badtarget", badTarget), act("nosettings", noSettings), act("mixed", mixed))

	errs := testService().Validate(context.Background(), w)
	got := strings.Join(fields(errs), ",")
	for _, want := range []string{
		"condition",
		"add_note.text", "add_note",
		"notify.recipient_id", "notify.target", "notify", "kind",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %s", want, got)
		}
	}

	byNode := map[string]int{}
	for _, e := range errs {
		byNode[e.NodeID]++
		if e.Field == "condition" && (e.Pos == nil || *e.Pos != 9) {
			t.Errorf("condition error pos = %v, want 9", e.Pos)
		}
	}
	for _, id := range []string{"cond", "bad", "mismatch", "missing", "norecip", "extra", "badtarget", "nosettings", "mixed"} {
		if byNode[id] == 0 {
			t.Errorf("no error attributed to node %s: %v", id, errs)
		}
	}
	if !strings.Contains(errs.Error(), "node mismatch: notify.recipient_id") {
		t.Errorf("error text: %s", errs.Error())
	}
}

func TestValidateMutatingActions(t *testing.T) {
	w := flow(
		models.Node{ID: "s", Kind: "set_status", Title: "s", Enabled: true, ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: 10}}},
		models.Node{ID: "p", Kind: "set_priority", Title: "p", Enabled: true, ActionSettings: models.ActionSettings{SetPriority: &models.SetPriorityAction{PriorityID: 3, PriorityName: "P3"}}},
		models.Node{ID: "o", Kind: "set_owner", Title: "o", Enabled: true, ActionSettings: models.ActionSettings{SetOwner: &models.SetOwnerAction{MemberID: 5}}},
		models.Node{ID: "r", Kind: "add_resource", Title: "r", Enabled: true, ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 5, Identifier: "stale"}}},
		models.Node{ID: "j", Kind: "patch", Title: "j", Enabled: true, ActionSettings: models.ActionSettings{Patch: &models.PatchAction{Ops: []byte(` [ {"op":"replace", "path":"severity", "value":"High"} ] `)}}},
		models.Node{ID: "n", Kind: "notify", Title: "n", Enabled: true, ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Target: models.TargetRoom, RecipientID: rid(1), Message: "{{event}} {{ticket.link}} for {{company}}"}}},
	)
	if errs := testService().Validate(context.Background(), w); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	n := w.Nodes
	if n[1].SetStatus.StatusName != "Escalated" {
		t.Errorf("status name not filled: %+v", n[1].SetStatus)
	}
	if n[3].SetOwner.Identifier != "jdoe" || n[4].AddResource.Identifier != "jdoe" {
		t.Errorf("member identifiers not filled: %+v %+v", n[3].SetOwner, n[4].AddResource)
	}
	if string(n[5].Patch.Ops) != `[{"op":"replace","path":"severity","value":"High"}]` {
		t.Errorf("patch ops not normalized: %s", n[5].Patch.Ops)
	}
}

func TestValidateMutatingActionErrors(t *testing.T) {
	w := flow(
		models.Node{ID: "1", Kind: "set_status", Title: "x", ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: 20}}}, // other board
		models.Node{ID: "2", Kind: "set_status", Title: "x", ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: 30}}}, // inactive
		models.Node{ID: "3", Kind: "set_status", Title: "x", ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: 99}}}, // missing
		models.Node{ID: "4", Kind: "set_status", Title: "x"}, // no settings
		models.Node{ID: "5", Kind: "set_priority", Title: "x", ActionSettings: models.ActionSettings{SetPriority: &models.SetPriorityAction{}}},                                                 // no id
		models.Node{ID: "6", Kind: "set_owner", Title: "x", ActionSettings: models.ActionSettings{SetOwner: &models.SetOwnerAction{MemberID: 6}}},                                               // deleted
		models.Node{ID: "7", Kind: "add_resource", Title: "x", ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 99}}},                                     // missing
		models.Node{ID: "8", Kind: "patch", Title: "x", ActionSettings: models.ActionSettings{Patch: &models.PatchAction{Ops: []byte(`{"op":"replace"}`)}}},                                     // not an array
		models.Node{ID: "9", Kind: "patch", Title: "x", ActionSettings: models.ActionSettings{Patch: &models.PatchAction{Ops: []byte(`[]`)}, SetStatus: &models.SetStatusAction{StatusID: 10}}}, // mixed
		models.Node{ID: "10", Kind: "notify", Title: "x", ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Target: models.TargetResourcesOwner, Message: "hi {{nope}}"}}},
	)
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
		w := flow(iff("c", c.cond))
		errs := testService().Validate(context.Background(), w)
		if c.want == "" {
			if len(errs) != 0 {
				t.Errorf("%q: unexpected errors %v", c.cond, errs)
			}
			continue
		}
		if len(errs) != 1 || errs[0].Field != "condition" || errs[0].NodeID != "c" || !strings.Contains(errs[0].Message, c.want) || errs[0].Pos == nil {
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
	if errs := svc.Validate(context.Background(), flow(iff("c", "contact/id in list 99"))); len(errs) != 0 {
		t.Errorf("nil Lists: unexpected errors %v", errs)
	}
}
