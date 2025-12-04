package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/thecoretg/ticketbot/internal/models"
)

var ErrNoRoomsForEmail = errors.New("no rooms found for this email")

type recipMap map[int]models.WebexRecipient

func (s *Service) getAllRecipients(ctx context.Context, t *models.FullTicket, rules []models.NotifierRule, isNew bool) ([]models.WebexRecipient, error) {
	// for connectwise member emails
	excludedEmails := make(map[string]struct{})
	includedEmails := make(map[string]struct{})
	recips := make(recipMap)

	if t.LatestNote != nil && t.LatestNote.Member != nil {
		excludedEmails[t.LatestNote.Member.PrimaryEmail] = struct{}{}
	}

	for _, m := range t.Resources {
		if m.PrimaryEmail != "" {
			if _, excl := excludedEmails[m.PrimaryEmail]; excl {
				continue
			}
			includedEmails[m.PrimaryEmail] = struct{}{}
		}
	}

	if isNew {
		for _, nr := range rules {
			r, err := s.WebexSvc.GetRecipient(ctx, nr.WebexRoomID)
			if err != nil {
				// TODO: once done testing, this should warn and not exit
				return nil, fmt.Errorf("getting room for notifier rule %d: %w", nr.ID, err)
			}

			recips[r.ID] = r
		}
	}

	for e := range includedEmails {
		r, err := s.WebexSvc.EnsurePersonRecipientByEmail(ctx, e)
		if err != nil {
			return nil, fmt.Errorf("ensuring recipient by email %s: %w", e, err)
		}

		recips[r.ID] = r
	}

	fwdProcd, err := s.processAllFwds(ctx, recips)
	if err != nil {
		// return pre-fwd processing
		slog.Error("forward processing failed; using original recipients", "ticket_id", t.Ticket.ID, "error", err)
		return recips.toSlice(), nil
	}

	return fwdProcd.toSlice(), nil
}

func (s *Service) processAllFwds(ctx context.Context, recips recipMap) (recipMap, error) {
	keys := make([]int, 0, len(recips))
	for id := range recips {
		keys = append(keys, id)
	}

	for _, id := range keys {
		r, ok := recips[id]
		if !ok {
			continue // was deleted somewhere in this loop
		}

		fwds, err := s.Forwards.ListBySourceRoomID(ctx, r.ID)
		if err != nil {
			// TODO: once done...you get the point
			return nil, fmt.Errorf("checking forwards for recipient id %d: %w", r.ID, err)
		}

		fwds = filterActiveFwds(fwds)
		if len(fwds) == 0 {
			continue
		}

		keep := false
		for _, f := range fwds {
			if f.UserKeepsCopy {
				keep = true
			}

			if _, ok := recips[f.DestID]; ok {
				continue
			}

			fm, err := s.WebexSvc.GetRecipient(ctx, f.DestID)
			if err != nil {
				// TODO: once done...
				return nil, fmt.Errorf("getting recipient info for forward destination %d: %w", f.DestID, err)
			}

			recips[f.DestID] = fm
		}

		if !keep {
			delete(recips, r.ID)
		}
	}

	return recips, nil
}

func filterActiveFwds(fwds []models.NotifierForward) []models.NotifierForward {
	var activeFwds []models.NotifierForward
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

func (m recipMap) toSlice() []models.WebexRecipient {
	out := make([]models.WebexRecipient, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}

	return out
}
