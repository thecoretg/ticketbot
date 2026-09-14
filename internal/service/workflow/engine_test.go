package workflow

import (
	"context"
	"errors"
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
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
	f.patchOps = append(f.patchOps, ops)
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

func notify(target models.NotifyTarget, recipient *int) models.Action {
	return models.Action{Kind: models.ActionNotify, Enabled: true, Notify: &models.NotifyAction{Target: target, RecipientID: recipient}}
}

func addNote(text string) models.Action {
	return models.Action{Kind: models.ActionAddNote, Enabled: true, AddNote: &models.AddNoteAction{Text: text, Internal: true}}
}

func skip() models.Action {
	return models.Action{Kind: models.ActionSkipNotify, Enabled: true}
}

func rule(name string, trig models.Trigger, cond string, stop bool, actions ...models.Action) models.Rule {
	return models.Rule{ID: name, Name: name, Enabled: true, Trigger: trig, Condition: cond, StopProcessing: stop, Actions: actions}
}

func wf(rules ...models.Rule) *models.Workflow {
	return &models.Workflow{ID: 7, BoardID: 1, Name: "test", Enabled: true, Rules: rules}
}

func run(t *testing.T, cw *fakeCW, w *models.Workflow, in Input) *Result {
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

func TestTriggers(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(
		rule("create", models.TriggerCreate, "", false, notify(models.TargetRoom, rid(1))),
		rule("update", models.TriggerUpdate, "", false, notify(models.TargetRoom, rid(2))),
		rule("both", models.TriggerBoth, "", false, notify(models.TargetRoom, rid(3))),
	)

	res := run(t, cw, w, Input{IsNew: true})
	if len(res.Notifies) != 2 || *res.Notifies[0].Target.RecipientID != 1 || *res.Notifies[1].Target.RecipientID != 3 {
		t.Fatalf("new ticket notifies = %+v", res.Notifies)
	}
	if res.Rules[1].Skipped != "trigger" {
		t.Errorf("update rule should be skipped by trigger: %+v", res.Rules[1])
	}

	res = run(t, cw, w, Input{IsNew: false})
	if len(res.Notifies) != 2 || *res.Notifies[0].Target.RecipientID != 2 {
		t.Fatalf("updated ticket notifies = %+v", res.Notifies)
	}
}

func TestConditionMatchAndNoMatch(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(
		rule("match", models.TriggerBoth, "summary contains 'help' and status/name = 'New'", false, notify(models.TargetRoom, rid(1))),
		rule("nomatch", models.TriggerBoth, "company/id = 999", false, notify(models.TargetRoom, rid(2))),
		rule("changed", models.TriggerBoth, "changed/status = true", false, notify(models.TargetRoom, rid(3))),
		rule("note", models.TriggerBoth, "latestNote/internalAnalysisFlag = true", false, notify(models.TargetRoom, rid(4))),
	)

	note := &psa.ServiceTicketNote{ID: 5, Text: "x", InternalAnalysisFlag: true}
	res := run(t, cw, w, Input{TriggerNote: note, Changes: []models.FieldChange{{Field: "status"}}})

	if len(res.Notifies) != 3 {
		t.Fatalf("expected 3 notifies, got %+v", res.Notifies)
	}
	if !res.Rules[0].Matched || res.Rules[1].Matched || !res.Rules[2].Matched || !res.Rules[3].Matched {
		t.Errorf("rule outcomes: %+v", res.Rules)
	}
}

func TestConditionErrorSkipsRule(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(
		rule("bad", models.TriggerBoth, "summary =", false, notify(models.TargetRoom, rid(1))),
		rule("good", models.TriggerBoth, "", false, notify(models.TargetRoom, rid(2))),
	)
	res := run(t, cw, w, Input{})
	if res.Rules[0].Err == nil || res.Rules[0].Matched {
		t.Errorf("bad rule should record an error: %+v", res.Rules[0])
	}
	if len(res.Notifies) != 1 || *res.Notifies[0].Target.RecipientID != 2 {
		t.Errorf("good rule should still run: %+v", res.Notifies)
	}
}

func TestStopProcessing(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(
		rule("first", models.TriggerBoth, "", true, notify(models.TargetRoom, rid(1))),
		rule("second", models.TriggerBoth, "", false, notify(models.TargetRoom, rid(2))),
	)
	res := run(t, cw, w, Input{})
	if len(res.Rules) != 1 || !res.Rules[0].Stopped {
		t.Fatalf("expected chain to stop after first rule: %+v", res.Rules)
	}
	if len(res.Notifies) != 1 {
		t.Errorf("notifies = %+v", res.Notifies)
	}

	// stop only applies when the rule matched
	w.Rules[0].Condition = "company/id = 999"
	res = run(t, cw, w, Input{})
	if len(res.Rules) != 2 || len(res.Notifies) != 1 || *res.Notifies[0].Target.RecipientID != 2 {
		t.Errorf("non-matching stop rule must not stop the chain: rules=%+v notifies=%+v", res.Rules, res.Notifies)
	}
}

func TestSkipNotifySuppressesLaterOnly(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(
		rule("early", models.TriggerBoth, "", false, notify(models.TargetRoom, rid(1))),
		rule("quiet", models.TriggerBoth, "", false, skip(), notify(models.TargetRoom, rid(2))),
		rule("late", models.TriggerBoth, "", false, notify(models.TargetResourcesOwner, nil)),
	)
	res := run(t, cw, w, Input{})
	if !res.Suppressed {
		t.Fatal("expected Suppressed")
	}
	if len(res.Notifies) != 1 || *res.Notifies[0].Target.RecipientID != 1 {
		t.Errorf("only the earlier notify should queue: %+v", res.Notifies)
	}
	want := []string{ResultQueued, ResultOK, ResultSkipped, ResultSkipped}
	got := results(res)
	if len(got) != len(want) {
		t.Fatalf("results = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("results = %v, want %v", got, want)
		}
	}
	if res.Actions[2].Reason != "suppressed" {
		t.Errorf("reason = %q", res.Actions[2].Reason)
	}
}

func TestAddNotePostsRefetchesAndLaterRulesSeeNewState(t *testing.T) {
	cw := &fakeCW{ticket: ticket(), apiMember: "ticketbot"}
	cw.onPost = func(*psa.ServiceTicketNote) {
		updated := ticket()
		updated.Status.ID, updated.Status.Name = 11, "Escalated"
		cw.ticket = updated
	}
	trigger := &psa.ServiceTicketNote{ID: 5, Text: "customer note"}

	w := wf(
		rule("note", models.TriggerBoth, "", false, addNote("Escalating this ticket")),
		rule("after", models.TriggerBoth, "status/name = 'Escalated'", false, notify(models.TargetRoom, rid(1))),
		rule("trigger note unchanged", models.TriggerBoth, "latestNote/text = 'customer note'", false, notify(models.TargetRoom, rid(2))),
	)
	res := run(t, cw, w, Input{TriggerNote: trigger})

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
	if len(res.Notifies) != 2 {
		t.Errorf("later rules should see refetched status and original trigger note: %+v", res.Notifies)
	}
}

func TestAddNoteDryRun(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(rule("note", models.TriggerBoth, "", false, addNote("hi"), notify(models.TargetRoom, rid(1))))
	res := run(t, cw, w, Input{DryRun: true})

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
	w := wf(rule("note", models.TriggerBoth, "", false, addNote("hi"), notify(models.TargetRoom, rid(1))))
	res := run(t, cw, w, Input{})
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

func TestDisabledRuleAndAction(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	off := notify(models.TargetRoom, rid(1))
	off.Enabled = false
	disabledRule := rule("off", models.TriggerBoth, "", false, notify(models.TargetRoom, rid(2)))
	disabledRule.Enabled = false

	w := wf(rule("on", models.TriggerBoth, "", false, off, notify(models.TargetRoom, rid(3))), disabledRule)
	res := run(t, cw, w, Input{})

	if res.Rules[1].Skipped != "disabled" {
		t.Errorf("rule should be skipped as disabled: %+v", res.Rules[1])
	}
	if res.Actions[0].Result != ResultSkipped || res.Actions[0].Reason != "disabled" {
		t.Errorf("action should be skipped as disabled: %+v", res.Actions[0])
	}
	if len(res.Notifies) != 1 || *res.Notifies[0].Target.RecipientID != 3 {
		t.Errorf("notifies = %+v", res.Notifies)
	}
}

func TestUnknownActionKind(t *testing.T) {
	cw := &fakeCW{ticket: ticket()}
	w := wf(rule("r", models.TriggerBoth, "", false, models.Action{Kind: "teleport", Enabled: true}))
	res := run(t, cw, w, Input{})
	if res.Actions[0].Result != ResultError {
		t.Errorf("unknown kind should error: %+v", res.Actions[0])
	}
}

func TestRunNilInputs(t *testing.T) {
	e := NewEngine(&fakeCW{})
	if _, err := e.Run(context.Background(), nil, Input{Ticket: ticket()}); err == nil {
		t.Error("nil workflow should error")
	}
	if _, err := e.Run(context.Background(), wf(), Input{}); err == nil {
		t.Error("nil ticket should error")
	}
}
