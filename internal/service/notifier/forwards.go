package notifier

import (
	"context"
	"fmt"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
)

func (s *Service) ListForwardsFull(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	return s.Forwards.ListAllFull(ctx)
}

func (s *Service) ListForwards(ctx context.Context) ([]*models.NotifierForward, error) {
	return s.Forwards.ListAll(ctx)
}

func (s *Service) ListForwardsBySourceID(ctx context.Context, id int) ([]*models.NotifierForward, error) {
	return s.Forwards.ListBySourceRoomID(ctx, id)
}

func (s *Service) GetForward(ctx context.Context, id int) (*models.NotifierForward, error) {
	return s.Forwards.Get(ctx, id)
}

func (s *Service) AddForward(ctx context.Context, f *models.NotifierForward) (*models.NotifierForward, error) {
	return s.Forwards.Insert(ctx, f)
}

func (s *Service) DeleteForward(ctx context.Context, id int) error {
	exists, err := s.Forwards.Exists(ctx, id)
	if err != nil {
		return fmt.Errorf("checking if notifier exists: %w", err)
	}

	if !exists {
		return models.ErrUserForwardNotFound
	}

	return s.Forwards.Delete(ctx, id)
}

func (s *Service) processAllFwds(ctx context.Context, in recipMap) (recipMap, error) {
	queue := make([]int, 0, len(in))
	seen := make(map[int]struct{})

	for id := range in {
		queue = append(queue, id)
	}

	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]

		// prevent A > B > C > A forward chains
		if _, done := seen[id]; done {
			continue
		}
		seen[id] = struct{}{}

		r, ok := in[id]
		if !ok {
			continue // was deleted somewhere in this loop
		}

		// get all forwards by the ID of the source recipient
		fwds, err := s.Forwards.ListBySourceRoomID(ctx, r.recipient.ID)
		if err != nil {
			// TODO: make this so it doesn't exit if only one fails. log it.
			return nil, fmt.Errorf("checking forwards for recipient id %d: %w", r.recipient.ID, err)
		}

		fwds = filterActiveFwds(fwds)
		if len(fwds) == 0 {
			continue
		}

		keep := false

		for _, f := range fwds {
			// all we need is one forward where the user is marked to keep a copy
			if f.UserKeepsCopy {
				keep = true
			}

			// if the forward destination recipient is in the map already without a forward,
			// there is no need to treat it as a forward; they are already in the ticket and would
			// get the notification regardless.
			if _, ok := in[f.DestID]; ok {
				continue
			}

			// get the destination recipient and add it to the recipients map
			fm, err := s.WebexSvc.GetRecipient(ctx, f.DestID)
			if err != nil {
				// TODO: once done...
				return nil, fmt.Errorf("getting recipient info for forward destination %d: %w", f.DestID, err)
			}

			in[f.DestID] = newRecipWithFwd(fm, r)
			queue = append(queue, f.DestID)
		}

		if !keep {
			// delete the source recipient so the don't get the notification
			delete(in, r.recipient.ID)
		}
	}

	return in, nil
}

// filterActiveFwds returns all forwards that are enabled if the current time is within the date range
func filterActiveFwds(fwds []*models.NotifierForward) []*models.NotifierForward {
	var activeFwds []*models.NotifierForward
	for _, f := range fwds {
		if f.Enabled && dateRangeActive(f.StartDate, f.EndDate) {
			activeFwds = append(activeFwds, f)
		}
	}

	return activeFwds
}

func dateRangeActive(start, end *time.Time) bool {
	now := time.Now()
	if start == nil {
		return false
	}

	if end == nil {
		return now.After(*start)
	}

	return now.After(*start) && now.Before(*end)
}
