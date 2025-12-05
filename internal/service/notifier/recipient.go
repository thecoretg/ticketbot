package notifier

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/internal/models"
)

var ErrNoRoomsForEmail = errors.New("no rooms found for this email")

type (
	recipData struct {
		recipient    models.WebexRecipient
		forwardChain []models.WebexRecipient
	}

	recipMap map[int]recipData
)

func newRecip(rec models.WebexRecipient) recipData {
	return recipData{recipient: rec}
}

func newRecipWithFwd(rec models.WebexRecipient, parent recipData) recipData {
	chain := append([]models.WebexRecipient{}, parent.forwardChain...)
	chain = append(chain, parent.recipient)
	return recipData{
		recipient:    rec,
		forwardChain: chain,
	}
}

func (r recipData) isNaturalRecipient() bool {
	return len(r.forwardChain) == 0
}

func (s *Service) getAllRecipients(ctx context.Context, t *models.FullTicket, rules []models.NotifierRule, isNew bool) ([]recipData, error) {
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
			slog.Debug("getAllRecipients: calling webexsvc.GetRecipient", "room_id", nr.WebexRecipientID)
			r, err := s.WebexSvc.GetRecipient(ctx, nr.WebexRecipientID)
			if err != nil {
				// TODO: once done testing, this should warn and not exit
				return nil, fmt.Errorf("getting room for notifier rule %d: %w", nr.ID, err)
			}

			recips[r.ID] = newRecip(r)
		}
	}

	for e := range includedEmails {
		r, err := s.WebexSvc.EnsurePersonRecipientByEmail(ctx, e)
		if err != nil {
			return nil, fmt.Errorf("ensuring recipient by email %s: %w", e, err)
		}

		recips[r.ID] = newRecip(r)
	}

	fwdProcd, err := s.processAllFwds(ctx, recips)
	if err != nil {
		// return pre-fwd processing
		slog.Error("forward processing failed; using original recipients", "ticket_id", t.Ticket.ID, "error", err)
		return recips.toSlice(), nil
	}

	return fwdProcd.toSlice(), nil
}

func (m recipMap) toSlice() []recipData {
	out := make([]recipData, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}

	return out
}
