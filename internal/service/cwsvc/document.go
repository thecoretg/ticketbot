package cwsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// TicketSnapshot returns the ticket and its latest note in ConnectWise shape for condition
// evaluation. It prefers the stored raw JSON and stored note; when no raw JSON exists (rows written
// before v2) it fetches live from ConnectWise. live reports which source was used.
func (s *Service) TicketSnapshot(ctx context.Context, id int) (t *psa.Ticket, note *psa.ServiceTicketNote, live bool, err error) {
	stored, err := s.Tickets.Get(ctx, id)
	if err != nil {
		return nil, nil, false, err
	}

	if len(stored.Raw) == 0 {
		f, err := s.FetchTicket(ctx, id)
		if err != nil {
			return nil, nil, true, fmt.Errorf("fetching ticket from connectwise: %w", err)
		}
		return f.Ticket, f.Note, true, nil
	}

	t = &psa.Ticket{}
	if err := json.Unmarshal(stored.Raw, t); err != nil {
		return nil, nil, false, fmt.Errorf("decoding stored ticket: %w", err)
	}

	if stored.LatestNoteID != nil {
		n, err := s.Notes.Get(ctx, *stored.LatestNoteID)
		if err != nil && !errors.Is(err, models.ErrTicketNoteNotFound) {
			return nil, nil, false, fmt.Errorf("getting stored note: %w", err)
		}
		if n != nil {
			note, err = s.noteToCW(ctx, n)
			if err != nil {
				return nil, nil, false, err
			}
		}
	}

	return t, note, false, nil
}

func (s *Service) noteToCW(ctx context.Context, n *models.TicketNote) (*psa.ServiceTicketNote, error) {
	out := &psa.ServiceTicketNote{
		ID:                   n.ID,
		TicketID:             n.TicketID,
		InternalAnalysisFlag: n.InternalAnalysisFlag,
	}
	if n.Content != nil {
		out.Text = *n.Content
	}

	if n.MemberID != nil {
		m, err := s.Members.Get(ctx, *n.MemberID)
		if err != nil && !errors.Is(err, models.ErrMemberNotFound) {
			return nil, fmt.Errorf("getting note member: %w", err)
		}
		if m != nil {
			out.Member.ID = m.ID
			out.Member.Identifier = m.Identifier
			out.Member.Name = memberName(m)
		}
	}

	if n.ContactID != nil {
		c, err := s.Contacts.Get(ctx, *n.ContactID)
		if err != nil && !errors.Is(err, models.ErrContactNotFound) {
			return nil, fmt.Errorf("getting note contact: %w", err)
		}
		if c != nil {
			out.Contact.ID = c.ID
			out.Contact.Name = c.FirstName
			if c.LastName != nil {
				out.Contact.Name += " " + *c.LastName
			}
		}
	}

	return out, nil
}
