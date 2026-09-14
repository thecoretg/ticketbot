package cwsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

const (
	defaultTicketPageSize = 50
	maxTicketPageSize     = 100
	detailEventLimit      = 100
)

// ListTickets returns a page of stored tickets with display names and ConnectWise links.
func (s *Service) ListTickets(ctx context.Context, f models.TicketFilter) (*models.TicketPage, error) {
	f.Page = max(f.Page, 1)
	if f.PageSize <= 0 {
		f.PageSize = defaultTicketPageSize
	}
	f.PageSize = min(f.PageSize, maxTicketPageSize)

	items, total, err := s.Tickets.ListPaged(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("listing tickets: %w", err)
	}

	for _, it := range items {
		it.CWURL = psa.InternalTicketLink(it.ID, s.CWCompanyID)
	}

	return &models.TicketPage{
		Items:    items,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	}, nil
}

// GetTicketDetail assembles a stored ticket, its related entities and its most recent events.
func (s *Service) GetTicketDetail(ctx context.Context, id int) (*models.TicketDetail, error) {
	t, err := s.Tickets.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	board, err := s.Boards.Get(ctx, t.BoardID)
	if err != nil {
		return nil, fmt.Errorf("getting board: %w", err)
	}

	status, err := s.Statuses.Get(ctx, t.StatusID)
	if err != nil {
		return nil, fmt.Errorf("getting status: %w", err)
	}

	company, err := s.Companies.Get(ctx, t.CompanyID)
	if err != nil {
		return nil, fmt.Errorf("getting company: %w", err)
	}

	d := &models.TicketDetail{
		Ticket: models.TicketListItem{
			Ticket:      *t,
			BoardName:   board.Name,
			StatusName:  status.Name,
			CompanyName: company.Name,
			CWURL:       psa.InternalTicketLink(t.ID, s.CWCompanyID),
		},
		Board:     *board,
		Status:    *status,
		Company:   *company,
		Resources: []*models.Member{},
		Events:    []*models.TicketEvent{},
	}

	if t.ContactID != nil {
		c, err := s.Contacts.Get(ctx, *t.ContactID)
		if err != nil && !errors.Is(err, models.ErrContactNotFound) {
			return nil, fmt.Errorf("getting contact: %w", err)
		}
		d.Contact = c
	}

	if t.OwnerID != nil {
		m, err := s.Members.Get(ctx, *t.OwnerID)
		if err != nil && !errors.Is(err, models.ErrMemberNotFound) {
			return nil, fmt.Errorf("getting owner: %w", err)
		}
		if m != nil {
			d.Owner = m
			d.Ticket.OwnerName = memberName(m)
		}
	}

	if t.Resources != nil && *t.Resources != "" {
		for _, ident := range resourceStringToSlice(*t.Resources) {
			m, err := s.Members.GetByIdentifier(ctx, ident)
			if err != nil {
				continue
			}
			d.Resources = append(d.Resources, m)
		}
	}

	if t.LatestNoteID != nil {
		n, err := s.Notes.Get(ctx, *t.LatestNoteID)
		if err == nil {
			d.LatestNote, err = repos.TicketNoteToFullTicketNote(ctx, n, s.Members, s.Contacts)
			if err != nil {
				return nil, fmt.Errorf("resolving latest note: %w", err)
			}
		} else if !errors.Is(err, models.ErrTicketNoteNotFound) {
			return nil, fmt.Errorf("getting latest note: %w", err)
		}
	}

	events, err := s.Events.ListByTicket(ctx, id, detailEventLimit, nil)
	if err != nil {
		return nil, fmt.Errorf("listing events: %w", err)
	}
	// ListByTicket is newest-first; the thread reads oldest-first.
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
	d.Events = events

	return d, nil
}

// GetTicketRaw returns the stored ConnectWise JSON for a ticket, or nil when none has been stored.
func (s *Service) GetTicketRaw(ctx context.Context, id int) (json.RawMessage, error) {
	t, err := s.Tickets.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	return t.Raw, nil
}

// ListTicketEvents pages backwards through a ticket's events, newest first.
func (s *Service) ListTicketEvents(ctx context.Context, id int, limit int, beforeID *int64) ([]*models.TicketEvent, error) {
	exists, err := s.Tickets.Exists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, models.ErrTicketNotFound
	}

	return s.Events.ListByTicket(ctx, id, limit, beforeID)
}

func memberName(m *models.Member) string {
	if m == nil {
		return ""
	}
	if m.FirstName == "" && m.LastName == "" {
		return m.Identifier
	}
	if m.LastName == "" {
		return m.FirstName
	}
	return m.FirstName + " " + m.LastName
}
