package syncsvc

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

func TestTrackerRefusesOverlappingRuns(t *testing.T) {
	tr := &tracker{}
	now := time.Now()
	if err := tr.start(&models.SyncPayload{CWBoards: true}, "a@x", now, nil); err != nil {
		t.Fatalf("first start: %v", err)
	}
	if err := tr.start(&models.SyncPayload{CWBoards: true}, "b@x", now, nil); !errors.Is(err, ErrSyncRunning) {
		t.Fatalf("second start = %v, want ErrSyncRunning", err)
	}
	if !tr.running() {
		t.Fatal("running() = false during a run")
	}

	tr.finish(now)
	if tr.running() {
		t.Fatal("running() = true after finish")
	}
	if err := tr.start(&models.SyncPayload{CWTickets: true}, "b@x", now, nil); err != nil {
		t.Fatalf("start after finish: %v", err)
	}
	if got := tr.snapshot().StartedBy; got != "b@x" {
		t.Errorf("StartedBy = %q, want the new run's", got)
	}
}

func TestTrackerPhasesFollowTheSelection(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWBoards: true, CWTickets: true}, "", time.Now(), nil)

	run := tr.snapshot()
	if len(run.Phases) != 2 || run.Phases[0].Name != models.SyncPhaseBoards || run.Phases[1].Name != models.SyncPhaseTickets {
		t.Fatalf("phases = %+v, want boards then tickets", run.Phases)
	}
	for _, ph := range run.Phases {
		if ph.State != models.SyncPhaseFetching {
			t.Errorf("%s starts %q, want fetching", ph.Name, ph.State)
		}
	}
	if tr.phase(models.SyncPhaseWebexRecipients) != nil {
		t.Error("unselected phase has a handle")
	}
}

func TestPhaseCountsFailuresAndCapsMessages(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWTickets: true}, "", time.Now(), nil)
	ph := tr.phase(models.SyncPhaseTickets)

	total := models.MaxSyncPhaseErrors + 10
	ph.counting("Processing tickets", total)
	ph.step(nil)
	for i := range total - 1 {
		ph.step(fmt.Errorf("ticket %d", i))
	}
	ph.done()

	got := tr.snapshot().Phases[0]
	if got.State != models.SyncPhaseDone || got.Done != total || got.Total != total {
		t.Errorf("phase = %s %d/%d, want done %d/%d", got.State, got.Done, got.Total, total, total)
	}
	if got.ErrorCount != total-1 {
		t.Errorf("ErrorCount = %d, want %d", got.ErrorCount, total-1)
	}
	if len(got.Errors) != models.MaxSyncPhaseErrors || got.Errors[0] != "ticket 0" {
		t.Errorf("Errors = %d messages starting %q, want the first %d", len(got.Errors), got.Errors[0], models.MaxSyncPhaseErrors)
	}
}

func TestPhaseFailRecordsTheReason(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWBoards: true}, "", time.Now(), nil)
	ph := tr.phase(models.SyncPhaseBoards)

	ph.counting("Syncing boards and statuses", 3)
	ph.step(nil)
	ph.fail("Rolled back, nothing saved", errors.New("committing tx: boom"))

	got := tr.snapshot().Phases[0]
	if got.State != models.SyncPhaseFailed || got.Label != "Rolled back, nothing saved" {
		t.Errorf("phase = %s %q, want failed and rolled back", got.State, got.Label)
	}
	if got.ErrorCount != 1 || got.Errors[0] != "committing tx: boom" {
		t.Errorf("errors = %d %v, want the commit error", got.ErrorCount, got.Errors)
	}
}

func TestSnapshotIsACopy(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWTickets: true, BoardIDs: []int{1}}, "", time.Now(), nil)
	ph := tr.phase(models.SyncPhaseTickets)
	ph.counting("Processing tickets", 2)
	ph.step(errors.New("first"))

	snap := tr.snapshot()
	ph.step(errors.New("second"))
	tr.finish(time.Now())

	if snap.Phases[0].Done != 1 || len(snap.Phases[0].Errors) != 1 || snap.FinishedAt != nil {
		t.Errorf("snapshot changed after it was taken: %+v", snap)
	}
	snap.Options.BoardIDs[0] = 99
	if tr.snapshot().Options.BoardIDs[0] != 1 {
		t.Error("editing the snapshot changed the run")
	}
}

func TestNilPhaseIsANoOp(t *testing.T) {
	var ph *phase
	ph.fetching("x")
	ph.counting("x", 1)
	ph.label("x")
	ph.step(errors.New("x"))
	ph.done()
	ph.fail("x", errors.New("x"))
}

func TestSnapshotEncodesEmptyListsAsArrays(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWBoards: true}, "", time.Now(), nil)
	b, err := json.Marshal(tr.snapshot())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), `"errors":[]`) {
		t.Errorf("a phase without errors encodes %s, want \"errors\":[]", b)
	}

	empty := &tracker{}
	_ = empty.start(&models.SyncPayload{}, "", time.Now(), nil)
	b, _ = json.Marshal(empty.snapshot())
	if !strings.Contains(string(b), `"phases":[]`) {
		t.Errorf("a run with no phases encodes %s, want \"phases\":[]", b)
	}
}

func TestTrackerReferencePhasesInPageOrder(t *testing.T) {
	tr := &tracker{}
	_ = tr.start(&models.SyncPayload{CWMembers: true, CWCompanies: true, CWContacts: true, CWTickets: true}, "", time.Now(), nil)

	run := tr.snapshot()
	var names []string
	for _, ph := range run.Phases {
		names = append(names, ph.Name)
	}
	want := []string{models.SyncPhaseMembers, models.SyncPhaseCompanies, models.SyncPhaseContacts, models.SyncPhaseTickets}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("phases = %v, want %v", names, want)
	}
	if run.Phases[2].Label != "Waiting for companies" {
		t.Errorf("contacts label = %q, want it to say it waits for companies", run.Phases[2].Label)
	}
}

func TestIDCondition(t *testing.T) {
	if got := idCondition([]int{3, 41, 900}); got != "id in (3,41,900)" {
		t.Errorf("idCondition = %q", got)
	}
}
