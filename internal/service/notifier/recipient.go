package notifier

import (
	"context"
	"log/slog"

	"github.com/thecoretg/ticketbot/models"
)

type (
	recipData struct {
		recipient    *models.WebexRecipient
		forwardChain []*models.WebexRecipient
		// simulated marks a recipient that was added by a simulation-mode notifier
		// rule or forward: the Webex message is not sent, but a skipped
		// ticket_notification is still recorded and a "Would notify …" journal event
		// is emitted.
		simulated bool
	}

	recipMap map[int]recipData
)

func newRecip(rec *models.WebexRecipient) recipData {
	return recipData{recipient: rec}
}

func newRecipWithFwd(rec *models.WebexRecipient, parent recipData) recipData {
	chain := append([]*models.WebexRecipient{}, parent.forwardChain...)
	chain = append(chain, parent.recipient)
	return recipData{
		recipient:    rec,
		forwardChain: chain,
	}
}

func (r recipData) isNaturalRecipient() bool {
	return len(r.forwardChain) == 0
}

// getAllRecipients resolves who a ticket notifies, per the board-setting model:
//
//   - New ticket:     each enabled setting's configured recipient (room or person)
//     PLUS the ticket's people (owner/resources).
//   - Updated ticket: the ticket's people only, and only when at least one enabled
//     setting has NotifyOnUpdate. The configured recipient is never
//     notified on updates.
//
// Simulation is authoritative: a recipient is simulated (recorded skipped, never
// sent) unless it is reachable via a non-simulated path. The ticket's people are
// simulated unless at least one of the settings that *govern* them for this event
// (all enabled settings on new; the NotifyOnUpdate settings on update) is real.
func (s *Service) getAllRecipients(ctx context.Context, t *models.FullTicket, rules []*models.NotifierRule, isNew bool) ([]recipData, error) {
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

	// if a ticket is closed it removes the resources, but not the owner. Most of the time the owner is also in resources,
	// but this safeguards edge cases.
	if t.Owner != nil && t.Owner.PrimaryEmail != "" {
		if _, excl := excludedEmails[t.Owner.PrimaryEmail]; !excl {
			includedEmails[t.Owner.PrimaryEmail] = struct{}{}
		}
	}

	// peopleGoverning are the settings that decide whether (and how — real vs
	// simulated) the ticket's people are notified for this event.
	var peopleGoverning []*models.NotifierRule
	if isNew {
		// New ticket: notify each setting's configured recipient (room or person).
		peopleGoverning = rules
		for _, nr := range rules {
			slog.Debug("getAllRecipients: calling webexsvc.GetRecipient", "recipient_id", nr.WebexRecipientID)
			r, err := s.WebexSvc.GetRecipient(ctx, nr.WebexRecipientID)
			if err != nil {
				slog.Error("getting stored webex recipient for notifier rule", "rule_id", nr.ID, "recipient_id", nr.WebexRecipientID, "error", err.Error())
				continue
			}

			// Real wins: never downgrade a recipient already added by a non-simulated
			// setting to simulated.
			if existing, ok := recips[r.ID]; ok && !existing.simulated {
				continue
			}
			rd := newRecip(r)
			rd.simulated = nr.SimulationMode
			recips[r.ID] = rd
		}
	} else {
		// Updated ticket: only the settings opted into update notifications govern
		// (and only the ticket's people are notified — never the configured recipient).
		for _, nr := range rules {
			if nr.NotifyOnUpdate {
				peopleGoverning = append(peopleGoverning, nr)
			}
		}
	}

	// The ticket's people are notified on new tickets, and on updates only when a
	// setting opted in. They are simulated unless a governing setting is real.
	if len(peopleGoverning) > 0 {
		peopleSimulated := true
		for _, nr := range peopleGoverning {
			if !nr.SimulationMode {
				peopleSimulated = false
				break
			}
		}

		for e := range includedEmails {
			r, err := s.WebexSvc.EnsurePersonRecipientByEmail(ctx, e)
			if err != nil {
				slog.Error("notifier: ensuring webex person by email", "ticket_id", t.Ticket.ID, "email", e, "error", err.Error())
				continue
			}

			// Real wins: a person already added by a non-simulated setting stays real.
			if existing, ok := recips[r.ID]; ok && !existing.simulated {
				continue
			}
			rd := newRecip(r)
			rd.simulated = peopleSimulated
			recips[r.ID] = rd
		}
	}

	fwdProcd, err := s.processAllFwds(ctx, recips)
	if err != nil {
		// return pre-fwd processing
		slog.Error("forward processing failed; using original recipients", "ticket_id", t.Ticket.ID, "error", err.Error())
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
