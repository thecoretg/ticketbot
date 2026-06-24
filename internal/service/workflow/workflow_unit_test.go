package workflow

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

func sampleTicket() *psa.Ticket {
	t := &psa.Ticket{ID: 42, Summary: "Printer broken"}
	t.Company.ID = 7
	t.Company.Identifier = "ACME"
	t.Company.Name = "Acme Corp"
	t.Board.ID = 3
	t.Owner.Identifier = "drosenthal"
	return t
}

func leaf(field, op, value string) models.ConditionNode {
	return models.ConditionNode{Condition: &models.Condition{Field: field, Operator: op, Value: value}}
}

func group(op string, children ...models.ConditionNode) models.ConditionNode {
	return models.ConditionNode{Group: &models.ConditionGroup{Operator: op, Children: children}}
}

func TestTicketUpdateRenderTemplates(t *testing.T) {
	tk := sampleTicket()
	p := &TicketUpdateParams{Ops: []PatchOpConfig{
		{Op: "replace", Path: "summary", Value: "[{{.Company.Identifier}}] {{.Summary}}"},
		{Op: "remove", Path: "owner"},
	}}
	if err := p.renderTemplates(tk); err != nil {
		t.Fatal(err)
	}
	if p.Ops[0].Value != "[ACME] Printer broken" {
		t.Fatalf("got %q", p.Ops[0].Value)
	}
	if p.Ops[1].Value != "" {
		t.Fatalf("remove op value should be untouched, got %q", p.Ops[1].Value)
	}
}

func TestTicketUpdateRenderMissingFieldErrors(t *testing.T) {
	tk := sampleTicket()
	p := &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "{{.Nonexistent}}"}}}
	if err := p.renderTemplates(tk); err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestTicketUpdateValidate(t *testing.T) {
	cases := []struct {
		name    string
		params  *TicketUpdateParams
		wantErr bool
	}{
		{"empty ops", &TicketUpdateParams{}, true},
		{"unknown field", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "nonsense", Value: "x"}}}, true},
		{"bad template", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "{{.Summary"}}}, true},
		{"unsupported op", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "add", Path: "summary", Value: "x"}}}, true},
		{"remove non-removable", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "remove", Path: "summary"}}}, true},
		{"remove removable", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "remove", Path: "owner"}}}, false},
		{"valid replace", &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "[{{.Company.Identifier}}] {{.Summary}}"}}}, false},
	}
	for _, c := range cases {
		err := c.params.validate()
		if (err != nil) != c.wantErr {
			t.Errorf("%s: got err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestTicketUpdateBuildValue(t *testing.T) {
	if v, err := updateFieldByPath["company"].buildValue("ACME"); err != nil || v.(map[string]any)["identifier"] != "ACME" {
		t.Fatalf("company identifier build: v=%v err=%v", v, err)
	}
	if v, err := updateFieldByPath["contact"].buildValue("99"); err != nil || v.(map[string]any)["id"] != 99 {
		t.Fatalf("contact id build: v=%v err=%v", v, err)
	}
	if v, err := updateFieldByPath["owner"].buildValue("drosenthal"); err != nil || v.(map[string]any)["identifier"] != "drosenthal" {
		t.Fatalf("owner identifier build: v=%v err=%v", v, err)
	}
	if v, err := updateFieldByPath["priority"].buildValue("High"); err != nil || v.(map[string]any)["name"] != "High" {
		t.Fatalf("priority name build: v=%v err=%v", v, err)
	}
	if v, err := updateFieldByPath["summary"].buildValue("hi"); err != nil || v.(string) != "hi" {
		t.Fatalf("summary text build: v=%v err=%v", v, err)
	}
	if _, err := updateFieldByPath["board"].buildValue("not-a-number"); err == nil {
		t.Fatal("expected error coercing non-numeric id")
	}
}

func TestTicketUpdateCheckDependencies(t *testing.T) {
	op := func(path, value string) PatchOpConfig { return PatchOpConfig{Op: "replace", Path: path, Value: value} }
	cases := []struct {
		name     string
		params   *TicketUpdateParams
		hasBoard bool
		wantErr  bool
	}{
		{"contact without company", &TicketUpdateParams{Ops: []PatchOpConfig{op("contact", "99")}}, false, true},
		{"contact with company", &TicketUpdateParams{Ops: []PatchOpConfig{op("company", "ACME"), op("contact", "99")}}, false, false},
		{"board without status", &TicketUpdateParams{Ops: []PatchOpConfig{op("board", "3")}}, false, true},
		{"board with status", &TicketUpdateParams{Ops: []PatchOpConfig{op("board", "3"), op("status", "5")}}, false, false},
		{"status no board context", &TicketUpdateParams{Ops: []PatchOpConfig{op("status", "5")}}, false, true},
		{"status satisfied by workflow board", &TicketUpdateParams{Ops: []PatchOpConfig{op("status", "5")}}, true, false},
		{"summary only", &TicketUpdateParams{Ops: []PatchOpConfig{op("summary", "x")}}, false, false},
	}
	for _, c := range cases {
		if err := c.params.checkDependencies(c.hasBoard); (err != nil) != c.wantErr {
			t.Errorf("%s: got err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestWorkflowApplies(t *testing.T) {
	tk := sampleTicket() // board 3, summary "Printer broken", company "Acme Corp"/"ACME"
	c := EvalCtx{Ticket: tk}
	cases := []struct {
		name   string
		wf     *models.Workflow
		isNew  bool
		expect bool
	}{
		{"matching board both", &models.Workflow{CwBoardID: 3, OnTicketAction: "both"}, false, true},
		{"other board", &models.Workflow{CwBoardID: 9, OnTicketAction: "both"}, true, false},
		{"new only on new", &models.Workflow{CwBoardID: 3, OnTicketAction: "new"}, true, true},
		{"new only on update", &models.Workflow{CwBoardID: 3, OnTicketAction: "new"}, false, false},
		{"updated only on update", &models.Workflow{CwBoardID: 3, OnTicketAction: "updated"}, false, true},
		{"summary contains match", &models.Workflow{CwBoardID: 3, OnTicketAction: "both",
			Root: &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("summary", "contains", "printer")}}}, true, true},
		{"summary contains miss", &models.Workflow{CwBoardID: 3, OnTicketAction: "both",
			Root: &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("summary", "contains", "laptop")}}}, true, false},
	}
	for _, tc := range cases {
		if got := workflowApplies(tc.wf, c, tc.isNew); got != tc.expect {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.expect)
		}
	}
}

func TestEvalGroupNested(t *testing.T) {
	tk := sampleTicket()
	note := &psa.ServiceTicketNote{Text: "Please escalate ASAP"}
	c := EvalCtx{Ticket: tk, LastNote: note}

	cases := []struct {
		name   string
		root   *models.ConditionGroup
		expect bool
	}{
		{"nil root matches", nil, true},
		{"empty root matches", &models.ConditionGroup{Operator: "and"}, true},
		{"and all match", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
			leaf("summary", "contains", "printer"), leaf("company_identifier", "equals", "ACME"),
		}}, true},
		{"and one miss", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
			leaf("summary", "contains", "printer"), leaf("company_identifier", "equals", "OTHER"),
		}}, false},
		{"or one match", &models.ConditionGroup{Operator: "or", Children: []models.ConditionNode{
			leaf("summary", "contains", "laptop"), leaf("company_identifier", "equals", "ACME"),
		}}, true},
		{"or none match", &models.ConditionGroup{Operator: "or", Children: []models.ConditionNode{
			leaf("summary", "contains", "laptop"), leaf("company_identifier", "equals", "OTHER"),
		}}, false},
		// (board match by name AND summary contains x) OR company is OTHER  => true via first group
		{"nested grouped", &models.ConditionGroup{Operator: "or", Children: []models.ConditionNode{
			group("and", leaf("summary", "contains", "broken"), leaf("company_name", "equals", "acme corp")),
			leaf("company_identifier", "equals", "OTHER"),
		}}, true},
		{"last note match", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
			leaf("last_note_text", "contains", "escalate"),
		}}, true},
		{"last note miss", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
			leaf("last_note_text", "contains", "resolved"),
		}}, false},
	}
	for _, tc := range cases {
		if got := evalGroup(tc.root, c); got != tc.expect {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.expect)
		}
	}
}

func TestLastNoteMissingNote(t *testing.T) {
	c := EvalCtx{Ticket: sampleTicket(), LastNote: nil}
	root := &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("last_note_text", "contains", "x")}}
	if evalGroup(root, c) {
		t.Fatal("last_note_text should not match when there is no note")
	}
}

func TestBoolCondition(t *testing.T) {
	mk := func(op string) *models.ConditionGroup {
		return &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("customer_updated_flag", op, "")}}
	}

	tk := sampleTicket()
	tk.CustomerUpdatedFlag = true
	c := EvalCtx{Ticket: tk}
	if !evalGroup(mk("is_true"), c) {
		t.Error("is_true should match when CustomerUpdatedFlag is true")
	}
	if evalGroup(mk("is_false"), c) {
		t.Error("is_false should not match when CustomerUpdatedFlag is true")
	}

	tk.CustomerUpdatedFlag = false
	if evalGroup(mk("is_true"), c) {
		t.Error("is_true should not match when CustomerUpdatedFlag is false")
	}
	if !evalGroup(mk("is_false"), c) {
		t.Error("is_false should match when CustomerUpdatedFlag is false")
	}
}

func TestValidateGroup(t *testing.T) {
	cases := []struct {
		name    string
		root    *models.ConditionGroup
		wantErr bool
	}{
		{"nil ok", nil, false},
		{"valid leaf", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("summary", "contains", "x")}}, false},
		{"unknown field", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("nonsense", "contains", "x")}}, true},
		{"unknown operator", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("summary", "matches_regex", "x")}}, true},
		{"bool field is_true", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("customer_updated_flag", "is_true", "")}}, false},
		{"bool field bad operator", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("customer_updated_flag", "contains", "x")}}, true},
		{"string field bool operator", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("summary", "is_true", "")}}, true},
		{"bad group operator", &models.ConditionGroup{Operator: "nand", Children: []models.ConditionNode{leaf("summary", "contains", "x")}}, true},
		{"nested valid", &models.ConditionGroup{Operator: "or", Children: []models.ConditionNode{
			group("and", leaf("summary", "contains", "x")),
		}}, false},
		{"both set", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
			{Condition: &models.Condition{Field: "summary", Operator: "contains", Value: "x"}, Group: &models.ConditionGroup{Operator: "and"}},
		}}, true},
		{"neither set", &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{{}}}, true},
	}
	for _, c := range cases {
		if err := validateGroup(c.root, 0); (err != nil) != c.wantErr {
			t.Errorf("%s: got err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestRegistryHasActions(t *testing.T) {
	reg := newRegistry()
	for _, a := range []string{"ticket_update", "add_note", "send_message", "skip_notifications", "add_resource", "add_email_cc"} {
		if _, ok := reg[a]; !ok {
			t.Errorf("missing action %q in registry", a)
		}
	}
}

func TestLastNoteConditions(t *testing.T) {
	tk := sampleTicket()
	note := &psa.ServiceTicketNote{Text: "x", DetailDescriptionFlag: true, InternalAnalysisFlag: true}
	note.Member.Name = "Jane Tech"
	c := EvalCtx{Ticket: tk, LastNote: note}

	cases := []struct {
		name   string
		node   models.ConditionNode
		expect bool
	}{
		{"sender contains", leaf("last_note_sender", "contains", "jane"), true},
		{"sender miss", leaf("last_note_sender", "equals", "bob"), false},
		// note is internal+discussion (not resolution)
		{"type any_of has internal", leaf("last_note_type", "is_any_of", "internal,resolution"), true},
		{"type any_of has discussion", leaf("last_note_type", "is_any_of", "discussion"), true},
		{"type any_of miss", leaf("last_note_type", "is_any_of", "resolution"), false},
		{"type none_of excludes resolution", leaf("last_note_type", "is_none_of", "resolution"), true},
		{"type none_of hits internal", leaf("last_note_type", "is_none_of", "internal"), false},
	}
	for _, tc := range cases {
		g := &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{tc.node}}
		if got := evalGroup(g, c); got != tc.expect {
			t.Errorf("%s: got %v want %v", tc.name, got, tc.expect)
		}
	}
}

func TestSummarizeConditions(t *testing.T) {
	if got := summarizeConditions(nil); got != "" {
		t.Fatalf("nil group should summarize to empty, got %q", got)
	}
	g := &models.ConditionGroup{Operator: "or", Children: []models.ConditionNode{
		group("and", leaf("summary", "contains", "printer"), leaf("status_name", "equals", "open")),
		leaf("company_identifier", "is_any_of", "ACME,GLOBEX"),
	}}
	got := summarizeConditions(g)
	want := `(Summary contains "printer" AND Status equals "open") OR Company ID is any of "ACME,GLOBEX"`
	if got != want {
		t.Fatalf("got  %q\nwant %q", got, want)
	}
}

func TestActionEvent(t *testing.T) {
	cases := []struct {
		actionType string
		change     Change
		wantText   string
		wantStatus string
	}{
		{"send_message", Change{Applied: true, To: "Helpdesk"}, "Sent notification to Helpdesk", "ok"},
		{"ticket_update", Change{Applied: true, Field: "summary"}, "Updated ticket (summary)", "ok"},
		{"ticket_update", Change{Applied: false}, "Ticket already up to date", "info"},
		{"add_resource", Change{Applied: true, To: "jtech"}, "Added resource jtech", "ok"},
		{"skip_notifications", Change{Applied: true}, "Skipped default notifications", "skip"},
		// Simulation mode: applied actions become "Would …" with skip status.
		{"send_message", Change{Applied: true, To: "Helpdesk", Simulated: true}, "Would send notification to Helpdesk", "skip"},
		{"ticket_update", Change{Applied: true, Field: "summary", Simulated: true}, "Would update ticket (summary)", "skip"},
		{"add_note", Change{Applied: true, Simulated: true}, "Would add note", "skip"},
		{"add_email_cc", Change{Applied: true, To: "a@b.com", Simulated: true}, "Would add email CC a@b.com", "skip"},
		{"skip_notifications", Change{Applied: true, Simulated: true}, "Would skip default notifications", "skip"},
		// A simulated no-op keeps its accurate "already …" phrasing and info status.
		{"ticket_update", Change{Applied: false, Simulated: true}, "Ticket already up to date", "info"},
	}
	for _, c := range cases {
		e := actionEvent(c.actionType, c.change)
		if e.Text != c.wantText || e.Status != c.wantStatus {
			t.Errorf("%s: got {%q,%q} want {%q,%q}", c.actionType, e.Text, e.Status, c.wantText, c.wantStatus)
		}
	}
}

// TestSimulateSkipsSideEffects verifies that in simulation mode an action reports
// what it would do without touching the CW/Webex clients. The Exec carries a nil
// CW client, so any attempt to actually mutate the ticket would panic.
func TestSimulateSkipsSideEffects(t *testing.T) {
	ctx := context.Background()
	ex := &Exec{Simulate: true} // nil CW/Webex on purpose

	t.Run("add_note", func(t *testing.T) {
		c, err := AddNote{}.Apply(ctx, ex, sampleTicket(), &AddNoteParams{Text: "hello"})
		if err != nil {
			t.Fatal(err)
		}
		if !c.Applied || c.Field != "note" {
			t.Fatalf("got %+v", c)
		}
	})

	t.Run("ticket_update", func(t *testing.T) {
		tk := sampleTicket() // summary "Printer broken"
		p := &TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "Changed"}}}
		c, err := TicketUpdate{}.Apply(ctx, ex, tk, p)
		if err != nil {
			t.Fatal(err)
		}
		if !c.Applied {
			t.Fatalf("expected applied change, got %+v", c)
		}
		if tk.Summary != "Printer broken" {
			t.Fatalf("simulation must not mutate the in-memory ticket, got %q", tk.Summary)
		}
	})

	t.Run("add_resource", func(t *testing.T) {
		tk := sampleTicket()
		c, err := AddResource{}.Apply(ctx, ex, tk, &AddResourceParams{MemberIdentifier: "jtech"})
		if err != nil {
			t.Fatal(err)
		}
		if !c.Applied || c.To != "jtech" {
			t.Fatalf("got %+v", c)
		}
	})
}

func TestIsAnyOfSingleValue(t *testing.T) {
	tk := sampleTicket() // company_identifier ACME
	c := EvalCtx{Ticket: tk}
	g := &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{
		leaf("company_identifier", "is_any_of", "foo,acme,bar"),
	}}
	if !evalGroup(g, c) {
		t.Fatal("is_any_of should match a single-valued field against a set")
	}
}

func TestValidateWorkflow(t *testing.T) {
	s := &Service{registry: newRegistry()}
	cfg := func(v any) json.RawMessage { b, _ := json.Marshal(v); return b }

	updateAction := models.Action{Type: "ticket_update", Config: cfg(TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "x"}}})}
	sendAction := models.Action{Type: "send_message", Config: cfg(SendMessageParams{RecipientID: 5, Text: "hi"})}

	cases := []struct {
		name    string
		wf      *models.Workflow
		wantErr bool
	}{
		{"no board", &models.Workflow{OnTicketAction: "both", Actions: []models.Action{updateAction}}, true},
		{"no actions", &models.Workflow{CwBoardID: 3, OnTicketAction: "both"}, true},
		{"bad on_ticket_action", &models.Workflow{CwBoardID: 3, OnTicketAction: "sometimes", Actions: []models.Action{updateAction}}, true},
		{"unknown action", &models.Workflow{CwBoardID: 3, OnTicketAction: "both", Actions: []models.Action{{Type: "nope"}}}, true},
		{"send without recipient", &models.Workflow{CwBoardID: 3, OnTicketAction: "both", Actions: []models.Action{{Type: "send_message", Config: cfg(SendMessageParams{Text: "hi"})}}}, true},
		{"valid update", &models.Workflow{CwBoardID: 3, OnTicketAction: "both", Actions: []models.Action{updateAction}}, false},
		{"valid multi-action", &models.Workflow{CwBoardID: 3, OnTicketAction: "both", Actions: []models.Action{updateAction, sendAction}}, false},
		{"bad condition tree", &models.Workflow{CwBoardID: 3, OnTicketAction: "both", Actions: []models.Action{updateAction},
			Root: &models.ConditionGroup{Operator: "and", Children: []models.ConditionNode{leaf("nonsense", "contains", "x")}}}, true},
	}
	for _, c := range cases {
		if err := s.validateWorkflow(c.wf); (err != nil) != c.wantErr {
			t.Errorf("%s: got err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

func TestValidateWorkflowDefaultsOnTicketAction(t *testing.T) {
	s := &Service{registry: newRegistry()}
	b, _ := json.Marshal(TicketUpdateParams{Ops: []PatchOpConfig{{Op: "replace", Path: "summary", Value: "x"}}})
	wf := &models.Workflow{CwBoardID: 3, Actions: []models.Action{{Type: "ticket_update", Config: b}}}
	if err := s.validateWorkflow(wf); err != nil {
		t.Fatal(err)
	}
	if wf.OnTicketAction != models.WorkflowOnBoth {
		t.Fatalf("expected on_ticket_action to default to both, got %q", wf.OnTicketAction)
	}
}

func TestAddNoteParamsUnmarshal(t *testing.T) {
	var p AddNoteParams
	if err := json.Unmarshal([]byte(`{"text":"hi {{.Company.Name}}","internal":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Internal || p.Text == "" {
		t.Fatalf("bad unmarshal: %+v", p)
	}
}

func TestUpdateFieldsCatalog(t *testing.T) {
	fields := UpdateFields()
	if len(fields) == 0 {
		t.Fatal("expected a non-empty update field catalog")
	}
	var sawOwnerRemovable bool
	for _, f := range fields {
		if f.Path == "owner" && f.AllowRemove {
			sawOwnerRemovable = true
		}
	}
	if !sawOwnerRemovable {
		t.Error("expected owner to be removable in the catalog")
	}
}

func TestTicketCard(t *testing.T) {
	tk := sampleTicket()
	note := &psa.ServiceTicketNote{Text: "needs a part"}
	note.Member.Name = "Jane Tech"
	card := ticketCard(tk, note, "thecore", 300)
	for _, want := range []string{"Printer broken", "Acme Corp", "Jane Tech", "> needs a part"} {
		if !strings.Contains(card, want) {
			t.Errorf("card missing %q:\n%s", want, card)
		}
	}
}
