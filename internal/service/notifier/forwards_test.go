package notifier

import (
	"context"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/models"
)

// fakeFwdRepo satisfies repos.NotifierForwardRepository via embedding; only the
// method processAllFwds calls is implemented.
type fakeFwdRepo struct {
	repos.NotifierForwardRepository
	bySource map[int][]*models.NotifierForwardFull
}

func (f *fakeFwdRepo) ListActiveBySourceRoomID(_ context.Context, id int) ([]*models.NotifierForwardFull, error) {
	return f.bySource[id], nil
}

// fakeRecipRepo satisfies repos.WebexRecipientRepository via embedding; only Get
// is implemented (used when a forward adds a destination).
type fakeRecipRepo struct {
	repos.WebexRecipientRepository
	byID map[int]*models.WebexRecipient
}

func (f *fakeRecipRepo) Get(_ context.Context, id int) (*models.WebexRecipient, error) {
	return f.byID[id], nil
}

func person(id int) *models.WebexRecipient {
	return &models.WebexRecipient{ID: id, Type: models.RecipientTypePerson}
}

func room(id int) *models.WebexRecipient {
	return &models.WebexRecipient{ID: id, Type: models.RecipientTypeRoom}
}

func newTestService(fwds map[int][]*models.NotifierForwardFull, recips map[int]*models.WebexRecipient) *Service {
	return &Service{
		Forwards: &fakeFwdRepo{bySource: fwds},
		WebexSvc: webexsvc.New(nil, &fakeRecipRepo{byID: recips}, nil),
	}
}

func fwd(dest int, sole, keep bool) *models.NotifierForwardFull {
	return &models.NotifierForwardFull{
		DestinationID:      dest,
		OnlyIfSoleResource: sole,
		UserKeepsCopy:      keep,
		Enabled:            true,
	}
}

func TestProcessAllFwds_SoleResource(t *testing.T) {
	tests := []struct {
		name    string
		natural []*models.WebexRecipient           // recipients present before forward processing
		fwds    map[int][]*models.NotifierForwardFull
		destDB  map[int]*models.WebexRecipient      // recipients GetRecipient can resolve
		wantIDs []int                               // expected recipient IDs after processing
	}{
		{
			name:    "sole-resource forward fires when source is only person",
			natural: []*models.WebexRecipient{person(1)},
			fwds:    map[int][]*models.NotifierForwardFull{1: {fwd(2, true, false)}},
			destDB:  map[int]*models.WebexRecipient{2: person(2)},
			wantIDs: []int{2}, // source dropped, dest added
		},
		{
			name:    "sole-resource forward suppressed when another person present",
			natural: []*models.WebexRecipient{person(1), person(3)},
			fwds:    map[int][]*models.NotifierForwardFull{1: {fwd(2, true, false)}},
			destDB:  map[int]*models.WebexRecipient{2: person(2)},
			wantIDs: []int{1, 3}, // no forward; source kept, dest not added
		},
		{
			name:    "non-sole forward still fires with another person present",
			natural: []*models.WebexRecipient{person(1), person(3)},
			fwds:    map[int][]*models.NotifierForwardFull{1: {fwd(2, false, false)}},
			destDB:  map[int]*models.WebexRecipient{2: person(2)},
			wantIDs: []int{2, 3}, // source 1 dropped, dest 2 added, 3 remains
		},
		{
			name:    "sole-resource forward with keep-copy keeps source",
			natural: []*models.WebexRecipient{person(1)},
			fwds:    map[int][]*models.NotifierForwardFull{1: {fwd(2, true, true)}},
			destDB:  map[int]*models.WebexRecipient{2: person(2)},
			wantIDs: []int{1, 2},
		},
		{
			name:    "rooms do not count as resources for sole-resource check",
			natural: []*models.WebexRecipient{person(1), room(9)},
			fwds:    map[int][]*models.NotifierForwardFull{1: {fwd(2, true, false)}},
			destDB:  map[int]*models.WebexRecipient{2: person(2)},
			wantIDs: []int{2, 9}, // source is sole person, so forward fires; room remains
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestService(tc.fwds, tc.destDB)

			in := make(recipMap)
			for _, r := range tc.natural {
				in[r.ID] = newRecip(r)
			}

			out, err := s.processAllFwds(context.Background(), in)
			if err != nil {
				t.Fatalf("processAllFwds: %v", err)
			}

			got := make(map[int]bool, len(out))
			for id := range out {
				got[id] = true
			}
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("recipient count = %d %v, want %d %v", len(got), keys(got), len(tc.wantIDs), tc.wantIDs)
			}
			for _, id := range tc.wantIDs {
				if !got[id] {
					t.Errorf("missing expected recipient %d; got %v", id, keys(got))
				}
			}
		})
	}
}

func keys(m map[int]bool) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
