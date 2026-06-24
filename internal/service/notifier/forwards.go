package notifier

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

func (s *Service) ListForwardsFull(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	return s.Forwards.ListAllFull(ctx)
}

func (s *Service) ListForwardsActive(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	return s.Forwards.ListAllActive(ctx)
}

func (s *Service) ListForwardsNotExpired(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	return s.Forwards.ListAllNotExpired(ctx)
}

func (s *Service) ListForwardsInactive(ctx context.Context) ([]*models.NotifierForwardFull, error) {
	return s.Forwards.ListAllInactive(ctx)
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

func (s *Service) UpdateForward(ctx context.Context, f *models.NotifierForward) (*models.NotifierForward, error) {
	if f == nil {
		return nil, errors.New("got nil forward")
	}

	exists, err := s.Forwards.Exists(ctx, f.ID)
	if err != nil {
		return nil, fmt.Errorf("checking if forward exists: %w", err)
	}

	if !exists {
		return nil, models.ErrUserForwardNotFound
	}

	return s.Forwards.Update(ctx, f)
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
		fwds, err := s.Forwards.ListActiveBySourceRoomID(ctx, r.recipient.ID)
		if err != nil {
			// TODO: make this so it doesn't exit if only one fails. log it.
			return nil, fmt.Errorf("checking forwards for recipient id %d: %w", r.recipient.ID, err)
		}

		if len(fwds) == 0 {
			continue
		}

		keep := false
		hasRealFwd := false

		for _, f := range fwds {
			// Simulated forward: show what it WOULD do without altering real delivery.
			// Add the destination as a simulated recipient (unless they're already a
			// real recipient, in which case they'd be notified anyway), and don't let
			// it influence the source keep/delete decision.
			if f.SimulationMode {
				if existing, ok := in[f.DestinationID]; ok && !existing.simulated {
					continue
				}
				fm, err := s.WebexSvc.GetRecipient(ctx, f.DestinationID)
				if err != nil {
					return nil, fmt.Errorf("getting recipient info for forward destination %d: %w", f.DestinationID, err)
				}
				rd := newRecipWithFwd(fm, r)
				rd.simulated = true
				in[f.DestinationID] = rd
				queue = append(queue, f.DestinationID)
				continue
			}

			hasRealFwd = true
			// all we need is one forward where the user is marked to keep a copy
			if f.UserKeepsCopy {
				keep = true
			}

			// if the forward destination recipient is already a real recipient,
			// there is no need to treat it as a forward; they are already in the ticket
			// and would get the notification regardless. (Real wins over simulated.)
			if existing, ok := in[f.DestinationID]; ok && !existing.simulated {
				continue
			}

			// get the destination recipient and add it to the recipients map
			fm, err := s.WebexSvc.GetRecipient(ctx, f.DestinationID)
			if err != nil {
				// TODO: once done...
				return nil, fmt.Errorf("getting recipient info for forward destination %d: %w", f.DestinationID, err)
			}

			rd := newRecipWithFwd(fm, r)
			rd.simulated = r.simulated // inherit: forwarding a simulated source is also simulated
			in[f.DestinationID] = rd
			queue = append(queue, f.DestinationID)
		}

		// Only real forwards suppress the source's own notification; a source with
		// only simulated forwards keeps receiving for real.
		if hasRealFwd && !keep {
			// delete the source recipient so they don't get the notification
			delete(in, r.recipient.ID)
		}
	}

	return in, nil
}
