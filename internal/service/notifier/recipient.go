package notifier

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thecoretg/ticketbot/models"
)

type (
	recipData struct {
		recipient    *models.WebexRecipient
		forwardChain []*models.WebexRecipient
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

// origin is the recipient the message was originally addressed to: the recipient itself for a
// natural recipient, or the first hop of the forward chain.
func (r recipData) origin() *models.WebexRecipient {
	if len(r.forwardChain) == 0 {
		return r.recipient
	}
	return r.forwardChain[0]
}

// resolveTarget returns the natural recipients for one notify channel, before forwards.
func (s *Service) resolveTarget(ctx context.Context, t *models.FullTicket, target models.NotifyAction) (recipMap, error) {
	recips := make(recipMap)

	switch target.Channel {
	case models.ChannelWebexRoom, models.ChannelWebexPerson:
		if target.RecipientID == nil {
			return nil, fmt.Errorf("notify channel %s has no recipient id", target.Channel)
		}
		r, err := s.WebexSvc.GetRecipient(ctx, *target.RecipientID)
		if err != nil {
			return nil, fmt.Errorf("getting recipient %d: %w", *target.RecipientID, err)
		}
		recips[r.ID] = newRecip(r)

	case models.ChannelResourcesOwner:
		for e := range s.resourceOwnerEmails(t) {
			r, err := s.WebexSvc.EnsurePersonRecipientByEmail(ctx, e)
			if err != nil {
				slog.Error("notifier: ensuring webex person by email", "ticket_id", t.Ticket.ID, "email", e, "error", err.Error())
				continue
			}
			recips[r.ID] = newRecip(r)
		}

	default:
		return nil, fmt.Errorf("unknown notify channel %q", target.Channel)
	}

	return recips, nil
}

// resourceOwnerEmails collects the ticket's resources and owner, excluding whoever wrote the
// note that triggered this notification.
func (s *Service) resourceOwnerEmails(t *models.FullTicket) map[string]struct{} {
	excluded := ""
	if t.LatestNote != nil && t.LatestNote.Member != nil {
		excluded = t.LatestNote.Member.PrimaryEmail
	}

	emails := make(map[string]struct{})
	for _, m := range t.Resources {
		if m.PrimaryEmail != "" && m.PrimaryEmail != excluded {
			emails[m.PrimaryEmail] = struct{}{}
		}
	}

	// if a ticket is closed it removes the resources, but not the owner. Most of the time the owner is also in resources,
	// but this safeguards edge cases.
	if t.Owner != nil && t.Owner.PrimaryEmail != "" && t.Owner.PrimaryEmail != excluded {
		emails[t.Owner.PrimaryEmail] = struct{}{}
	}

	return emails
}

// applyForwards runs forward rules over the union of natural recipients. On failure the original
// set is returned so a broken forward never blocks a notification.
func (s *Service) applyForwards(ctx context.Context, t *models.FullTicket, recips recipMap) recipMap {
	// notes flagged for internal analysis never redirect through a public-only forward
	noteIsInternal := t.LatestNote != nil && t.LatestNote.InternalAnalysisFlag

	work := make(recipMap, len(recips))
	for id, r := range recips {
		work[id] = r
	}

	fwdProcd, err := s.processAllFwds(ctx, work, noteIsInternal)
	if err != nil {
		slog.Error("forward processing failed; using original recipients", "ticket_id", t.Ticket.ID, "error", err.Error())
		return recips
	}

	return fwdProcd
}

func (m recipMap) toSlice() []recipData {
	out := make([]recipData, 0, len(m))
	for _, r := range m {
		out = append(out, r)
	}

	return out
}
