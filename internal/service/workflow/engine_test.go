package workflow

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/thecoretg/ticketbot/internal/cwquery"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

// fakeCW records writes and lets tests mutate the ticket returned by the next refetch.
type fakeCW struct {
	ticket    *psa.Ticket
	note      *psa.ServiceTicketNote
	posted    []*psa.ServiceTicketNote
	postErr   error
	onPost    func(*psa.ServiceTicketNote) // mutate ticket/note to simulate CW side effects
	nextNote  int
	getCalls  int
	patchOps  [][]psa.PatchOp
	patchErr  error
	onPatch   func([]psa.PatchOp) // mutate ticket to simulate the CW response
	apiMember string
}

func (f *fakeCW) GetTicket(_ context.Context, _ int, _ map[string]string) (*psa.Ticket, error) {
	f.getCalls++
	return f.ticket, nil
}

func (f *fakeCW) GetMostRecentTicketNote(_ context.Context, _ int) (*psa.ServiceTicketNote, error) {
	if f.note == nil {
		return nil, psa.ErrNotFound
	}
	return f.note, nil
}

func (f *fakeCW) PatchTicket(_ context.Context, _ int, ops []psa.PatchOp) (*psa.Ticket, error) {
	if f.patchErr != nil {
		return nil, f.patchErr
	}
	f.patchOps = append(f.patchOps, ops)
	if f.onPatch != nil {
		f.onPatch(ops)
	}
	return f.ticket, nil
}

func (f *fakeCW) PostServiceTicketNote(_ context.Context, n *psa.ServiceTicketNote, ticketID int) (*psa.ServiceTicketNote, error) {
	if f.postErr != nil {
		return nil, f.postErr
	}
	f.nextNote++
	posted := *n
	posted.ID = 1000 + f.nextNote
	posted.TicketID = ticketID
	posted.Member.Identifier = f.apiMember
	f.posted = append(f.posted, &posted)
	f.note = &posted
	if f.onPost != nil {
		f.onPost(&posted)
	}
	return &posted, nil
}

func ticket() *psa.Ticket {
	t := &psa.Ticket{ID: 42, Summary: "Printer HELP"}
	t.Board.ID = 1
	t.Status.ID, t.Status.Name = 10, "New"
	t.Company.ID = 100
	return t
}

func rid(id int) *int { return &id }

func notify(target models.NotifyChannel, recipient *int) models.Action {
	return models.Action{Kind: models.ActionNotify, Enabled: true, ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Channel: target, RecipientID: recipient}}}
}

func addNote(text string) models.Action {
	return models.Action{Kind: models.ActionAddNote, Enabled: true, ActionSettings: models.ActionSettings{AddNote: &models.AddNoteAction{Text: text, Internal: true}}}
}

func skip() models.Action {
	return models.Action{Kind: models.ActionSkipNotify, Enabled: true}
}

func setStatus(id int) models.Action {
	return models.Action{Kind: models.ActionSetStatus, Enabled: true, ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: id, StatusName: "S"}}}
}

// --- graph builders ---

func trig(id string, x float64, events ...models.TriggerEvent) models.Node {
	if len(events) == 0 {
		events = []models.TriggerEvent{models.TriggerCreated, models.TriggerUpdated}
	}
	return models.Node{ID: id, Kind: models.NodeTrigger, Title: id, Enabled: true, X: x, Events: events}
}

func iff(id, cond string) models.Node {
	return models.Node{ID: id, Kind: models.NodeIf, Title: id, Enabled: true, Condition: cond}
}

func act(id string, a models.Action) models.Node {
	return models.Node{ID: id, Kind: models.NodeKind(a.Kind), Title: id, Enabled: a.Enabled, ActionSettings: a.ActionSettings}
}

func edge(from string, port models.Port, to string) models.Edge {
	return models.Edge{ID: from + "/" + string(port) + ">" + to, From: from, To: to, Port: port}
}

func graph(nodes []models.Node, edges ...models.Edge) *models.Workflow {
	return &models.Workflow{ID: 7, BoardID: 1, Name: "test", Enabled: true, Nodes: nodes, Edges: edges}
}

// rule and legacy build a v1 chain and upgrade it, so the engine tests double as upgrade tests:
// every legacy semantic below must survive the conversion.
func rule(name string, trig models.Trigger, cond string, stop bool, actions ...models.Action) models.Rule {
	return models.Rule{ID: name, Name: name, Enabled: true, Trigger: trig, Condition: cond, StopProcessing: stop, Actions: actions}
}

func legacy(rules ...models.Rule) *models.Workflow {
	nodes, edges := models.UpgradeRules(rules)
	return graph(nodes, edges...)
}

func exec(t *testing.T, cw *fakeCW, w *models.Workflow, in Input) *Result {
	t.Helper()
	if in.Ticket == nil {
		in.Ticket = cw.ticket
	}
	res, err := NewEngine(cw).Run(context.Background(), w, in)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func results(res *Result) []string {
	out := make([]string, 0, len(res.Actions))
	for _, a := range res.Actions {
		out = append(out, a.Result)
	}
	return out
}

func recipients(res *Result) []int {
	out := make([]int, 0, len(res.Notifies))
	for _, n := range res.Notifies {
		if n.Action.RecipientID != nil {
			out = append(out, *n.Action.RecipientID)
		} else {
			out = append(out, 0)
		}
	}
	return out
}

func step(res *Result, id string) *StepOutcome {
	for i := range res.Steps {
		if res.Steps[i].Step.NodeID == id {
			return &res.Steps[i]
		}
	}
	return nil
}

func path(res *Result) []string {
	out := make([]string, 0, len(res.Steps))
	for _, s := range res.Steps {
		out = append(out, s.Step.NodeID)
	}
	return out
}

func matched(res *Result, id string) bool {
	s := step(res, id)
	return s != nil && s.Matched != nil && *s.Matched
}

func eq[T comparable](t *testing.T, what string, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", what, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", what, got, want)
		}
	}
}

// --- graph semantics ---

func TestEveryListeningTriggerFires(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t-upd", 300, models.TriggerUpdated), act("n2", notify(models.ChannelWebexRoom, rid(2))),
		trig("t-new", 0, models.TriggerCreated), act("n1", notify(models.ChannelWebexRoom, rid(1))),
		trig("t-both", 600), act("n3", notify(models.ChannelWebexRoom, rid(3))),
		{ID: "t-off", Kind: models.NodeTrigger, Title: "off", Enabled: false, X: 900, Events: []models.TriggerEvent{models.TriggerCreated}},
		act("n4", notify(models.ChannelWebexRoom, rid(4))),
	},
		edge("t-new", models.PortOut, "n1"), edge("t-upd", models.PortOut, "n2"),
		edge("t-both", models.PortOut, "n3"), edge("t-off", models.PortOut, "n4"),
	)

	res := exec(t, cw, w, Input{IsNew: true})
	eq(t, "new ticket recipients", recipients(res), []int{1, 3})
	eq(t, "new ticket path", path(res), []string{"t-new", "n1", "t-both", "n3"})
	if res.Event != models.TriggerCreated {
		t.Errorf("event = %q", res.Event)
	}
	if s := step(res, "n1"); s.Trigger != "t-new" || s.Via != "t-new/out>n1" || s.No != 2 {
		t.Errorf("step n1 = %+v", *s)
	}

	res = exec(t, cw, w, Input{IsNew: false})
	eq(t, "updated ticket recipients", recipients(res), []int{2, 3})

	// a document with no trigger for the event does nothing
	none := graph([]models.Node{trig("t", 0, models.TriggerCreated)})
	res = exec(t, cw, none, Input{IsNew: false})
	if len(res.Steps) != 0 || len(res.Actions) != 0 {
		t.Errorf("no listening trigger should produce no steps: %+v", res.Steps)
	}
}

func TestBranchesFollowThePortAndRejoinOnce(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t", 0),
		iff("crit", "summary contains 'help'"),
		act("yes", notify(models.ChannelWebexRoom, rid(1))),
		act("no", notify(models.ChannelWebexRoom, rid(2))),
		act("join", addNote("done")),
	},
		edge("t", models.PortOut, "crit"),
		edge("crit", models.PortYes, "yes"), edge("crit", models.PortNo, "no"),
		edge("yes", models.PortOut, "join"), edge("no", models.PortOut, "join"),
	)

	res := exec(t, cw, w, Input{})
	eq(t, "path", path(res), []string{"t", "crit", "yes", "join"})
	eq(t, "recipients", recipients(res), []int{1})
	if s := step(res, "crit"); s.Port != models.PortYes || !matched(res, "crit") {
		t.Errorf("if step = %+v", *s)
	}
	if s := step(res, "join"); s.Port != "" {
		t.Errorf("last step should have no exit port: %+v", *s)
	}

	cw.ticket.Summary = "quiet"
	res = exec(t, cw, w, Input{})
	eq(t, "else path", path(res), []string{"t", "crit", "no", "join"})
	eq(t, "else recipients", recipients(res), []int{2})
	if len(cw.posted) != 2 {
		t.Errorf("join ran %d times, want once per run", len(cw.posted))
	}
}

func TestNodeReachedByTwoTriggersRunsOnce(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("a", 0), trig("b", 300),
		act("shared", notify(models.ChannelWebexRoom, rid(1))),
		act("after", notify(models.ChannelWebexRoom, rid(2))),
	},
		edge("a", models.PortOut, "shared"), edge("b", models.PortOut, "shared"),
		edge("shared", models.PortOut, "after"),
	)
	res := exec(t, cw, w, Input{})
	eq(t, "recipients", recipients(res), []int{1, 2})
	eq(t, "path", path(res), []string{"a", "shared", "after", "b", "shared"})
	if s := res.Steps[4]; s.Skipped != SkippedJoined || s.Trigger != "b" {
		t.Errorf("second visit should be recorded as joined: %+v", s)
	}
}

func TestSkipNotifySilencesOnlyItsOwnPath(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t", 0),
		iff("c", "id = 42"),
		act("quiet", skip()), act("silenced", notify(models.ChannelWebexRoom, rid(1))),
		act("loud", notify(models.ChannelWebexRoom, rid(2))),
		trig("t2", 300), act("other", notify(models.ChannelWebexRoom, rid(3))),
	},
		edge("t", models.PortOut, "c"),
		edge("c", models.PortYes, "quiet"), edge("quiet", models.PortOut, "silenced"),
		edge("c", models.PortNo, "loud"),
		edge("t2", models.PortOut, "other"),
	)
	res := exec(t, cw, w, Input{})
	eq(t, "recipients", recipients(res), []int{3})
	eq(t, "results", results(res), []string{ResultOK, ResultSkipped, ResultQueued})
	if res.Actions[1].Reason != "suppressed" {
		t.Errorf("reason = %q", res.Actions[1].Reason)
	}
}

func TestDisabledNodesPassThrough(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	off := notify(models.ChannelWebexRoom, rid(1))
	off.Enabled = false
	offIf := iff("c", "id = 42")
	offIf.Enabled = false
	w := graph([]models.Node{
		trig("t", 0), offIf,
		act("yes", notify(models.ChannelWebexRoom, rid(2))),
		act("off", off), act("on", notify(models.ChannelWebexRoom, rid(3))),
	},
		edge("t", models.PortOut, "c"),
		edge("c", models.PortYes, "yes"), edge("c", models.PortNo, "off"),
		edge("off", models.PortOut, "on"),
	)
	res := exec(t, cw, w, Input{})
	eq(t, "path", path(res), []string{"t", "c", "off", "on"})
	if s := step(res, "c"); s.Skipped != SkippedDisabled || s.Port != models.PortNo || s.Matched != nil {
		t.Errorf("disabled if should take the no port: %+v", *s)
	}
	if s := step(res, "off"); s.Skipped != SkippedDisabled {
		t.Errorf("disabled action step = %+v", *s)
	}
	if res.Actions[0].Result != ResultSkipped || res.Actions[0].Reason != "disabled" {
		t.Errorf("action should be skipped as disabled: %+v", res.Actions[0])
	}
	eq(t, "recipients", recipients(res), []int{3})
}

func TestConditionErrorTakesTheElsePort(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t", 0), iff("bad", "summary ="),
		act("yes", notify(models.ChannelWebexRoom, rid(1))), act("no", notify(models.ChannelWebexRoom, rid(2))),
	},
		edge("t", models.PortOut, "bad"), edge("bad", models.PortYes, "yes"), edge("bad", models.PortNo, "no"),
	)
	res := exec(t, cw, w, Input{})
	if s := step(res, "bad"); s.Err == nil || s.Matched != nil || s.Port != models.PortNo {
		t.Errorf("bad condition step = %+v", *s)
	}
	eq(t, "recipients", recipients(res), []int{2})
}

func TestCyclesStopAtTheStepCap(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	// a → b → a: validation forbids this, the engine must still terminate
	w := graph([]models.Node{
		trig("t", 0), act("a", skip()), act("b", skip()),
	},
		edge("t", models.PortOut, "a"), edge("a", models.PortOut, "b"), edge("b", models.PortOut, "a"),
	)
	res := exec(t, cw, w, Input{})
	eq(t, "path", path(res), []string{"t", "a", "b", "a"})
	if res.Steps[3].Skipped != SkippedJoined {
		t.Errorf("revisit should be joined: %+v", res.Steps[3])
	}

	// a huge chain of distinct nodes hits MaxSteps
	nodes := []models.Node{trig("t", 0)}
	var edges []models.Edge
	prev := "t"
	for i := 0; i < MaxSteps+10; i++ {
		id := "n" + strings.Repeat("x", 1) + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + string(rune('0'+i/260))
		nodes = append(nodes, act(id, skip()))
		edges = append(edges, edge(prev, models.PortOut, id))
		prev = id
	}
	res = exec(t, cw, graph(nodes, edges...), Input{})
	if len(res.Steps) != MaxSteps {
		t.Errorf("steps = %d, want %d", len(res.Steps), MaxSteps)
	}
}

func TestUnknownNodeKindEndsThePath(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t", 0), {ID: "x", Kind: "teleport", Title: "x", Enabled: true}, act("n", notify(models.ChannelWebexRoom, rid(1))),
	}, edge("t", models.PortOut, "x"), edge("x", models.PortOut, "n"))
	res := exec(t, cw, w, Input{})
	eq(t, "path", path(res), []string{"t", "x"})
	if step(res, "x").Err == nil {
		t.Error("unknown kind should record an error")
	}
}

func TestRunNilInputs(t *testing.T) {
	e := NewEngine(&fakeCW{})
	if _, err := e.Run(context.Background(), nil, Input{Ticket: ticket()}); err == nil {
		t.Error("nil workflow should error")
	}
	if _, err := e.Run(context.Background(), graph(nil), Input{}); err == nil {
		t.Error("nil ticket should error")
	}
}

// --- legacy chains, upgraded ---

func TestLegacyTriggers(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("create", models.TriggerCreate, "", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("update", models.TriggerUpdate, "", false, notify(models.ChannelWebexRoom, rid(2))),
		rule("both", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(3))),
	)

	res := exec(t, cw, w, Input{IsNew: true})
	eq(t, "new ticket recipients", recipients(res), []int{1, 3})
	if matched(res, "update") {
		t.Errorf("update rule must not match a new ticket: %+v", *step(res, "update"))
	}

	res = exec(t, cw, w, Input{IsNew: false})
	eq(t, "updated ticket recipients", recipients(res), []int{2, 3})
}

func TestLegacyConditionMatchAndNoMatch(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("match", models.TriggerBoth, "summary contains 'help' and status/name = 'New'", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("nomatch", models.TriggerBoth, "company/id = 999", false, notify(models.ChannelWebexRoom, rid(2))),
		rule("changed", models.TriggerBoth, "changed/status = true", false, notify(models.ChannelWebexRoom, rid(3))),
		rule("note", models.TriggerBoth, "latestNote/internalAnalysisFlag = true", false, notify(models.ChannelWebexRoom, rid(4))),
	)

	note := &psa.ServiceTicketNote{ID: 5, Text: "x", InternalAnalysisFlag: true}
	res := exec(t, cw, w, Input{TriggerNote: note, Changes: []models.FieldChange{{Field: "status"}}})

	eq(t, "recipients", recipients(res), []int{1, 3, 4})
	if !matched(res, "match") || matched(res, "nomatch") || !matched(res, "changed") || !matched(res, "note") {
		t.Errorf("steps: %+v", res.Steps)
	}
}

func TestLegacyConditionErrorSkipsRule(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("bad", models.TriggerBoth, "summary =", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("good", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(2))),
	)
	res := exec(t, cw, w, Input{})
	if s := step(res, "bad"); s.Err == nil || matched(res, "bad") {
		t.Errorf("bad rule should record an error: %+v", *s)
	}
	eq(t, "recipients", recipients(res), []int{2})
}

func TestLegacyStopProcessing(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("first", models.TriggerBoth, "", true, notify(models.ChannelWebexRoom, rid(1))),
		rule("second", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(2))),
	)
	res := exec(t, cw, w, Input{})
	if step(res, "second") != nil {
		t.Fatalf("chain should end after the stop rule: %v", path(res))
	}
	eq(t, "recipients", recipients(res), []int{1})

	// stop only applies when the rule matched
	w = legacy(
		rule("first", models.TriggerBoth, "company/id = 999", true, notify(models.ChannelWebexRoom, rid(1))),
		rule("second", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(2))),
	)
	res = exec(t, cw, w, Input{})
	if step(res, "second") == nil {
		t.Errorf("non-matching stop rule must not stop the chain: %v", path(res))
	}
	eq(t, "recipients", recipients(res), []int{2})
}

func TestLegacySkipNotifySuppressesLaterOnly(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("early", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("quiet", models.TriggerBoth, "", false, skip(), notify(models.ChannelWebexRoom, rid(2))),
		rule("late", models.TriggerBoth, "", false, notify(models.ChannelResourcesOwner, nil)),
	)
	res := exec(t, cw, w, Input{})
	eq(t, "recipients", recipients(res), []int{1})
	eq(t, "results", results(res), []string{ResultQueued, ResultOK, ResultSkipped, ResultSkipped})
	if res.Actions[2].Reason != "suppressed" {
		t.Errorf("reason = %q", res.Actions[2].Reason)
	}
}

func TestLegacyDisabledRule(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	disabled := rule("off", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(2)))
	disabled.Enabled = false
	w := legacy(rule("on", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(1))), disabled,
		rule("after", models.TriggerBoth, "", false, notify(models.ChannelWebexRoom, rid(3))))
	res := exec(t, cw, w, Input{})
	if s := step(res, "off"); s == nil || s.Skipped != SkippedDisabled {
		t.Errorf("rule should be skipped as disabled: %+v", s)
	}
	eq(t, "recipients", recipients(res), []int{1, 3})
}

// --- actions ---

// Notes are posted after the walks, so a ConnectWise-side effect of the note (here a status
// change) is visible in the refetched result but not to later nodes of the same run.
func TestAddNotePostsAfterTheWalksAndRefetches(t *testing.T) {
	cw := &fakeCW{ticket: ticket(), apiMember: "ticketbot"}
	cw.onPost = func(*psa.ServiceTicketNote) {
		updated := ticket()
		updated.Status.ID, updated.Status.Name = 11, "Escalated"
		cw.ticket = updated
	}
	trigger := &psa.ServiceTicketNote{ID: 5, Text: "customer note"}

	w := legacy(
		rule("note", models.TriggerBoth, "", false, addNote("Escalating this ticket")),
		rule("after", models.TriggerBoth, "status/name = 'Escalated'", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("trigger note unchanged", models.TriggerBoth, "latestNote/text = 'customer note'", false, notify(models.ChannelWebexRoom, rid(2))),
	)
	res := exec(t, cw, w, Input{TriggerNote: trigger})

	if len(cw.posted) != 1 || cw.posted[0].Text != "Escalating this ticket" || !cw.posted[0].InternalAnalysisFlag {
		t.Fatalf("posted = %+v", cw.posted)
	}
	if res.CWWrites != 1 || res.LearnedAPIMember != "ticketbot" {
		t.Errorf("writes=%d learned=%q", res.CWWrites, res.LearnedAPIMember)
	}
	if res.Actions[0].Result != ResultOK || res.Actions[0].Output["note_id"] != 1001 {
		t.Errorf("add_note outcome: %+v", res.Actions[0])
	}
	if res.Ticket.Status.Name != "Escalated" {
		t.Errorf("ticket not refetched: %+v", res.Ticket.Status)
	}
	if res.LatestNote == nil || res.LatestNote.ID != 1001 {
		t.Errorf("latest note should be the posted note: %+v", res.LatestNote)
	}
	if res.TriggerNote != trigger {
		t.Error("trigger note must be preserved")
	}
	eq(t, "recipients", recipients(res), []int{2})
}

func TestAddNoteDryRun(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(rule("note", models.TriggerBoth, "", false, addNote("hi"), notify(models.ChannelWebexRoom, rid(1))))
	res := exec(t, cw, w, Input{DryRun: true})

	if len(cw.posted) != 0 || cw.getCalls != 0 {
		t.Fatal("dry run must not touch ConnectWise")
	}
	if res.Actions[0].Result != ResultWouldRun || res.Actions[0].Output["text"] != "hi" {
		t.Errorf("dry run outcome: %+v", res.Actions[0])
	}
	// notify intents are still queued in dry run; the sender decides not to send
	if len(res.Notifies) != 1 || res.Actions[1].Result != ResultQueued {
		t.Errorf("notify should still queue in dry run: %+v", res.Actions[1])
	}
}

func TestAddNoteError(t *testing.T) {
	cw := &fakeCW{ticket: ticket(), postErr: errors.New("boom")}
	w := legacy(rule("note", models.TriggerBoth, "", false, addNote("hi"), notify(models.ChannelWebexRoom, rid(1))))
	res := exec(t, cw, w, Input{})
	if res.Actions[0].Result != ResultError || res.Actions[0].Err == nil {
		t.Errorf("expected error outcome: %+v", res.Actions[0])
	}
	if len(res.Notifies) != 1 {
		t.Error("a failed action must not stop later actions")
	}
	if res.CWWrites != 0 {
		t.Error("failed post is not a write")
	}
}

func TestUnknownActionKind(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	// a kind the model does not know is not an action node either; a known kind with no
	// settings is the action-level error path
	w := graph([]models.Node{trig("t", 0), {ID: "n", Kind: "notify", Title: "n", Enabled: true}}, edge("t", models.PortOut, "n"))
	res := exec(t, cw, w, Input{})
	if len(res.Actions) != 1 || res.Actions[0].Result != ResultError {
		t.Errorf("missing settings should error: %+v", res.Actions)
	}
}

func TestNewNoteAndOldValuesInConditions(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(
		rule("reply", models.TriggerUpdate, "newNote = true", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("left new", models.TriggerUpdate, "changed/status = true and old/status/name = 'Assigned'", false, notify(models.ChannelWebexRoom, rid(2))),
		rule("wrong old", models.TriggerUpdate, "old/status/name = 'Closed'", false, notify(models.ChannelWebexRoom, rid(3))),
	)

	res := exec(t, cw, w, Input{NewNote: true, Changes: []models.FieldChange{
		{Field: "status", Old: ticketdiff.Ref{ID: 8, Name: "Assigned"}, New: ticketdiff.Ref{ID: 10, Name: "New"}},
	}})
	eq(t, "recipients", recipients(res), []int{1, 2})

	// a field-only update is not a new note
	res = exec(t, cw, w, Input{Changes: []models.FieldChange{{Field: "summary", Old: "a", New: "b"}}})
	if len(res.Notifies) != 0 {
		t.Errorf("field-only update should not match newNote or old/status: %+v", res.Notifies)
	}
}

func TestIsNewInConditions(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := graph([]models.Node{
		trig("t", 0), iff("new", "isNew = true"),
		act("y", notify(models.ChannelWebexRoom, rid(1))), act("n", notify(models.ChannelWebexRoom, rid(2))),
	}, edge("t", models.PortOut, "new"), edge("new", models.PortYes, "y"), edge("new", models.PortNo, "n"))
	eq(t, "new", recipients(exec(t, cw, w, Input{IsNew: true})), []int{1})
	eq(t, "updated", recipients(exec(t, cw, w, Input{IsNew: false})), []int{2})
}

func TestSetStatusPatchesAndLaterNodesSeeIt(t *testing.T) {
	original := ticket()
	cw := &fakeCW{ticket: original}
	cw.onPatch = func(ops []psa.PatchOp) {
		updated := ticket()
		updated.Status.ID, updated.Status.Name = 11, "Escalated"
		updated.Info.UpdatedBy = "ticketbot"
		cw.ticket = updated
	}
	// The later node sees the queued status (id and the action's name) before anything is written.
	w := legacy(
		rule("escalate", models.TriggerBoth, "", false, setStatus(11)),
		rule("after", models.TriggerBoth, "status/id = 11 and status/name = 'S'", false, notify(models.ChannelWebexRoom, rid(1))),
	)
	res := exec(t, cw, w, Input{})

	if original.Status.ID != 10 {
		t.Fatal("the caller's ticket must not be mutated by queued writes")
	}

	if len(cw.patchOps) != 1 || cw.patchOps[0][0].Path != "status/id" || cw.patchOps[0][0].Value != 11 {
		t.Fatalf("patch ops = %+v", cw.patchOps)
	}
	if res.Actions[0].Result != ResultOK || res.CWWrites != 1 || res.LearnedAPIMember != "ticketbot" {
		t.Errorf("outcome=%+v writes=%d learned=%q", res.Actions[0], res.CWWrites, res.LearnedAPIMember)
	}
	if res.Ticket.Status.Name != "Escalated" || len(res.Notifies) != 1 {
		t.Errorf("later node should see patched status: %+v %+v", res.Ticket.Status, res.Notifies)
	}
	if cw.getCalls != 0 {
		t.Error("patch response should replace the ticket without a refetch")
	}
}

func TestMutatingActionsSkipWhenAlreadyApplied(t *testing.T) {
	tk := ticket()
	tk.Priority.ID = 4
	tk.Owner.ID = 77
	tk.Resources = "jdoe, asmith"
	cw := &fakeCW{ticket: tk}

	w := legacy(rule("r", models.TriggerBoth, "", false,
		setStatus(10),
		models.Action{Kind: models.ActionSetPriority, Enabled: true, ActionSettings: models.ActionSettings{SetPriority: &models.SetPriorityAction{PriorityID: 4}}},
		models.Action{Kind: models.ActionSetOwner, Enabled: true, ActionSettings: models.ActionSettings{SetOwner: &models.SetOwnerAction{MemberID: 77}}},
		models.Action{Kind: models.ActionAddResource, Enabled: true, ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 1, Identifier: "JDOE"}}},
	))
	res := exec(t, cw, w, Input{})

	for i, a := range res.Actions {
		if a.Result != ResultSkipped {
			t.Errorf("action %d should be skipped: %+v", i, a)
		}
	}
	if len(cw.patchOps) != 0 {
		t.Errorf("no patches expected: %+v", cw.patchOps)
	}
}

func TestAddResourceAppendsIdentifier(t *testing.T) {
	tk := ticket()
	tk.Resources = "jdoe"
	cw := &fakeCW{ticket: tk}
	w := legacy(rule("r", models.TriggerBoth, "", false,
		models.Action{Kind: models.ActionAddResource, Enabled: true, ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 2, Identifier: "asmith"}}},
	))
	res := exec(t, cw, w, Input{})

	if res.Actions[0].Result != ResultOK || len(cw.patchOps) != 1 {
		t.Fatalf("outcome=%+v ops=%+v", res.Actions[0], cw.patchOps)
	}
	if op := cw.patchOps[0][0]; op.Path != "resources" || op.Value != "jdoe,asmith" {
		t.Errorf("op = %+v", op)
	}
}

func TestPatchActionAndDryRun(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	raw := []byte(`[{"op":"replace","path":"severity","value":"High"},{"op":"remove","path":"contact"}]`)
	w := legacy(rule("r", models.TriggerBoth, "", false,
		models.Action{Kind: models.ActionPatch, Enabled: true, ActionSettings: models.ActionSettings{Patch: &models.PatchAction{Ops: raw}}},
		setStatus(11),
	))

	res := exec(t, cw, w, Input{DryRun: true})
	if res.Actions[0].Result != ResultWouldRun || res.Actions[1].Result != ResultWouldRun || len(cw.patchOps) != 0 {
		t.Fatalf("dry run must not patch: %+v", res.Actions)
	}

	if cw.ticket.Status.ID != 10 {
		t.Fatal("dry run must leave the caller's ticket untouched")
	}

	// Live: both actions merge into one PATCH, in canvas order.
	res = exec(t, cw, w, Input{})
	if res.Actions[0].Result != ResultOK || res.Actions[1].Result != ResultOK || len(cw.patchOps) != 1 || len(cw.patchOps[0]) != 3 {
		t.Fatalf("live run: %+v ops=%+v", res.Actions, cw.patchOps)
	}
	if cw.patchOps[0][1].Op != "remove" || cw.patchOps[0][1].Path != "contact" || cw.patchOps[0][2].Path != "status/id" {
		t.Errorf("ops = %+v", cw.patchOps[0])
	}
	if res.CWWrites != 1 {
		t.Errorf("one batched PATCH is one write, got %d", res.CWWrites)
	}
}

func TestConflictingWritesLastWinsAndIsRecorded(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := legacy(rule("r", models.TriggerBoth, "", false, setStatus(11), setStatus(12)))
	res := exec(t, cw, w, Input{})

	if len(cw.patchOps) != 1 || len(cw.patchOps[0]) != 1 || cw.patchOps[0][0].Value != 12 {
		t.Fatalf("ops = %+v, want one status/id=12", cw.patchOps)
	}
	if res.Actions[0].Result != ResultSuperseded || res.Actions[1].Result != ResultOK {
		t.Errorf("results = %v", results(res))
	}
	if len(res.Conflicts) != 1 || res.Conflicts[0].Path != "status/id" {
		t.Errorf("conflicts = %+v", res.Conflicts)
	}
}

func TestTwoAddResourcesMergeWithoutConflict(t *testing.T) {
	tk := ticket()
	tk.Resources = "jdoe"
	cw := &fakeCW{ticket: tk}
	w := legacy(rule("r", models.TriggerBoth, "", false,
		models.Action{Kind: models.ActionAddResource, Enabled: true, ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 2, Identifier: "asmith"}}},
		models.Action{Kind: models.ActionAddResource, Enabled: true, ActionSettings: models.ActionSettings{AddResource: &models.AddResourceAction{MemberID: 3, Identifier: "bkim"}}},
	))
	res := exec(t, cw, w, Input{})

	if len(cw.patchOps) != 1 || len(cw.patchOps[0]) != 1 || cw.patchOps[0][0].Value != "asmith,jdoe,bkim" {
		t.Fatalf("ops = %+v", cw.patchOps)
	}
	eq(t, "results", results(res), []string{ResultOK, ResultOK})
	if len(res.Conflicts) != 0 {
		t.Errorf("merging resources is not a conflict: %+v", res.Conflicts)
	}
}

func TestPatchErrorRestoresTheTicketAndFailsEveryQueuedAction(t *testing.T) {
	cw := &fakeCW{ticket: ticket(), patchErr: errors.New("cw down")}
	w := legacy(rule("r", models.TriggerBoth, "", false, setStatus(11), notify(models.ChannelWebexRoom, rid(1))))
	res := exec(t, cw, w, Input{})

	if res.Actions[0].Result != ResultError || res.Actions[0].Err == nil {
		t.Errorf("queued write should fail: %+v", res.Actions[0])
	}
	if res.Ticket.Status.ID != 10 {
		t.Errorf("unwritten state must not leak into the result: %+v", res.Ticket.Status)
	}
	if len(res.Notifies) != 1 || res.CWWrites != 0 {
		t.Errorf("notifies=%d writes=%d", len(res.Notifies), res.CWWrites)
	}
}

type refuseLimiter struct{ calls int }

func (l *refuseLimiter) Reserve(int, int) error { l.calls++; return errors.New("write cap reached") }

func TestRateCapRefusesWritesAndFlagsTheRun(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	lim := &refuseLimiter{}
	e := NewEngine(cw)
	e.Limiter = lim
	w := legacy(rule("r", models.TriggerBoth, "", false, setStatus(11), addNote("hi")))
	res, err := e.Run(context.Background(), w, Input{Ticket: cw.ticket})
	if err != nil {
		t.Fatal(err)
	}
	if !res.RateCapped || len(cw.patchOps) != 0 || len(cw.posted) != 0 {
		t.Fatalf("capped=%v patches=%d notes=%d", res.RateCapped, len(cw.patchOps), len(cw.posted))
	}
	eq(t, "results", results(res), []string{ResultError, ResultError})
	if lim.calls != 2 {
		t.Errorf("limiter asked %d times, want 2 (one PATCH, one note)", lim.calls)
	}
}

func TestPatchActionRejectsBadOps(t *testing.T) {
	for _, raw := range []string{``, `{}`, `[]`, `[{"op":"move","path":"x","value":1}]`, `[{"op":"replace","path":"","value":1}]`, `[{"op":"replace","path":"x"}]`} {
		if _, err := DecodePatchOps([]byte(raw)); err == nil {
			t.Errorf("%s should be rejected", raw)
		}
	}
	ops, err := DecodePatchOps([]byte(`[{"op":"remove","path":"contact"}]`))
	if err != nil || len(ops) != 1 {
		t.Errorf("remove without value should be accepted: %v %v", ops, err)
	}
}

func TestPatchError(t *testing.T) {
	cw := &fakeCW{ticket: ticket(), patchErr: errors.New("boom")}
	w := legacy(rule("r", models.TriggerBoth, "", false, setStatus(11), notify(models.ChannelWebexRoom, rid(1))))
	res := exec(t, cw, w, Input{})
	if res.Actions[0].Result != ResultError || res.CWWrites != 0 || len(res.Notifies) != 1 {
		t.Errorf("failed patch should record error and not stop later actions: %+v", res.Actions)
	}
}

type fakeLists struct {
	lists cwquery.Lists
	err   error
	calls int
}

func (f *fakeLists) Memberships(context.Context) (cwquery.Lists, error) {
	f.calls++
	return f.lists, f.err
}

func TestInListConditions(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	cw.ticket.Contact.ID = 123
	lists := &fakeLists{lists: cwquery.Lists{3: cwquery.NewListSet(123), 4: cwquery.NewListSet(100)}}
	w := legacy(
		rule("contact in", models.TriggerBoth, "contact/id in list 3", false, notify(models.ChannelWebexRoom, rid(1))),
		rule("contact not in", models.TriggerBoth, "contact/id not in list 3", false, notify(models.ChannelWebexRoom, rid(2))),
		rule("company in", models.TriggerBoth, "company/id in list 4", false, notify(models.ChannelWebexRoom, rid(3))),
		rule("missing list", models.TriggerBoth, "company/id in list 99", false, notify(models.ChannelWebexRoom, rid(4))),
		rule("missing negated", models.TriggerBoth, "company/id not in list 99", false, notify(models.ChannelWebexRoom, rid(5))),
	)

	eng := NewEngine(cw)
	eng.Lists = lists
	res, err := eng.Run(context.Background(), w, Input{Ticket: cw.ticket})
	if err != nil {
		t.Fatal(err)
	}
	eq(t, "recipients", recipients(res), []int{1, 3, 5})
	if lists.calls != 1 {
		t.Errorf("loader calls = %d, want 1", lists.calls)
	}

	// no list refs: loader never consulted
	lists.calls = 0
	if _, err := eng.Run(context.Background(), legacy(rule("plain", models.TriggerBoth, "id = 42", false)), Input{Ticket: cw.ticket}); err != nil {
		t.Fatal(err)
	}
	if lists.calls != 0 {
		t.Errorf("loader called %d times for a workflow without lists", lists.calls)
	}

	// loader failure is recorded on the node and later nodes still run
	lists.err = errors.New("db down")
	res, _ = eng.Run(context.Background(), w, Input{Ticket: cw.ticket})
	if s := step(res, "contact in"); s.Err == nil || !strings.Contains(s.Err.Error(), "loading lists") {
		t.Errorf("step err = %v", s.Err)
	}
	if step(res, "missing negated") == nil {
		t.Errorf("later nodes should still run: %v", path(res))
	}

	// nil loader: in list is false, not in list is true
	res = exec(t, cw, w, Input{})
	eq(t, "nil loader recipients", recipients(res), []int{2, 5})
}
