package ticketbot

import (
	"errors"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

func TestBotTriggeredRun(t *testing.T) {
	botEdited := &models.FullTicket{}
	botEdited.Ticket.UpdatedBy = new("wtbot")
	humanEdited := &models.FullTicket{}
	humanEdited.Ticket.UpdatedBy = new("jdoe")

	cases := []struct {
		name           string
		workflowRan    bool
		wfBotTriggered bool
		full           *models.FullTicket
		want           bool
	}{
		// The bug: workflow ran its actions (so the synced ticket now shows the bot as
		// editor), but the run was human-triggered → must NOT be treated as bot.
		{"workflow ran, human trigger, bot-edited after", true, false, botEdited, false},
		{"workflow ran, bot trigger", true, true, botEdited, true},
		{"no workflow, bot-edited (loop echo)", false, false, botEdited, true},
		{"no workflow, human-edited", false, false, humanEdited, false},
		{"no workflow, no ticket", false, false, nil, false},
	}
	for _, c := range cases {
		if got := botTriggeredRun(c.workflowRan, c.wfBotTriggered, c.full, "wtbot"); got != c.want {
			t.Errorf("%s: got %v want %v", c.name, got, c.want)
		}
	}
}

func TestBuildRun(t *testing.T) {
	now := time.Now()
	okEvent := models.JournalEvent{Text: "Sent notification to X", Status: models.JournalOK}
	errEvent := models.JournalEvent{Text: "boom", Status: models.JournalError}
	infoEvent := models.JournalEvent{Text: "Matched workflow", Status: models.JournalInfo}
	simEvent := models.JournalEvent{Text: "Would notify X", Status: models.JournalSkip, Simulated: true}

	cases := []struct {
		name        string
		isNew       bool
		events      []models.JournalEvent
		err         error
		wantTrigger string
		wantOutcome string
		wantError   bool
	}{
		{"new completed", true, []models.JournalEvent{okEvent}, nil, models.TriggerNew, models.OutcomeCompleted, false},
		{"updated nothing", false, []models.JournalEvent{infoEvent}, nil, models.TriggerUpdated, models.OutcomeNothingToDo, false},
		{"empty nothing", false, nil, nil, models.TriggerUpdated, models.OutcomeNothingToDo, false},
		{"event error", false, []models.JournalEvent{okEvent, errEvent}, nil, models.TriggerUpdated, models.OutcomeWithErrors, true},
		{"fatal error", true, nil, errors.New("sync failed"), models.TriggerNew, models.OutcomeWithErrors, true},
		{"simulated", false, []models.JournalEvent{simEvent}, nil, models.TriggerUpdated, models.OutcomeSimulated, false},
		{"error beats simulated", false, []models.JournalEvent{simEvent, errEvent}, nil, models.TriggerUpdated, models.OutcomeWithErrors, true},
	}
	for _, c := range cases {
		r := buildRun(now, c.isNew, false, c.events, c.err)
		if r.Trigger != c.wantTrigger || r.Outcome != c.wantOutcome || r.HadError != c.wantError {
			t.Errorf("%s: got {trigger:%q outcome:%q err:%v} want {%q %q %v}",
				c.name, r.Trigger, r.Outcome, r.HadError, c.wantTrigger, c.wantOutcome, c.wantError)
		}
	}

	// A fatal error with no events should synthesize an error line.
	r := buildRun(now, false, false, nil, errors.New("kaboom"))
	if len(r.Events) != 1 || r.Events[0].Status != models.JournalError {
		t.Fatalf("expected a synthesized error event, got %+v", r.Events)
	}
}
