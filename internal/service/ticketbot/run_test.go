package ticketbot

import (
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

func TestRunSummaryVerdicts(t *testing.T) {
	base := time.Date(2026, 9, 21, 12, 0, 0, 0, time.UTC)
	now := base
	newTestRun := func() *run {
		r := newRun(42, models.SourceWebhook, func() time.Time { return now })
		r.wf = &models.Workflow{ID: 7, Name: "Help Desk"}
		r.boardID = 1
		r.res = &workflow.Result{Event: models.TriggerUpdated}
		return r
	}

	if r := newRun(1, models.SourceWebhook, nil); r.summary() != nil {
		t.Fatal("no workflow means no summary")
	}

	r := newTestRun()
	r.res.Steps = []workflow.StepOutcome{{}, {}}
	r.res.Actions = []workflow.ActionOutcome{{Result: workflow.ResultOK}}
	r.res.CWWrites = 1
	r.notifs = []notifier.Outcome{{Result: notifier.ResultSent}, {Result: notifier.ResultNoRecipients}}
	now = base.Add(1500 * time.Millisecond)
	s := r.summary()
	if s.Outcome != models.OutcomeClean || s.Steps != 2 || s.Writes != 1 || s.NotifSent != 1 || s.NotifNone != 1 || s.DurationMs != 1500 || *s.WorkflowID != 7 {
		t.Errorf("clean summary = %+v", s)
	}

	r = newTestRun()
	if s := r.summary(); s.Outcome != models.OutcomeNoTrigger {
		t.Errorf("no steps should be no_trigger, got %s", s.Outcome)
	}

	r = newTestRun()
	r.res.Steps = []workflow.StepOutcome{{}}
	r.notifs = []notifier.Outcome{{Result: notifier.ResultNoRecipients}}
	if s := r.summary(); s.Outcome != models.OutcomeNobodyNotified {
		t.Errorf("only empty notifies should be nobody_notified, got %s", s.Outcome)
	}

	r = newTestRun()
	r.res.Steps = []workflow.StepOutcome{{}}
	r.res.Actions = []workflow.ActionOutcome{{Result: workflow.ResultError}}
	if s := r.summary(); s.Outcome != models.OutcomeErrors || s.Errors != 1 {
		t.Errorf("failed action should be errors, got %+v", s)
	}

	r = newTestRun()
	r.res.Steps = []workflow.StepOutcome{{}}
	r.add(models.EventError, models.ErrorPayload{Stage: "notify", Error: "boom"})
	if s := r.summary(); s.Outcome != models.OutcomeErrors || s.Errors != 1 {
		t.Errorf("error event should count, got %+v", s)
	}
}
