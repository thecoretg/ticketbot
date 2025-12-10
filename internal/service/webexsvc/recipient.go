package webexsvc

import (
	"context"
	"fmt"
	"sort"

	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

func (s *Service) ListRecipient(ctx context.Context) ([]models.WebexRecipient, error) {
	return s.Recipients.List(ctx)
}

func (s *Service) ListRooms(ctx context.Context) ([]models.WebexRecipient, error) {
	return s.Recipients.ListRooms(ctx)
}

func (s *Service) ListPeople(ctx context.Context) ([]models.WebexRecipient, error) {
	return s.Recipients.ListPeople(ctx)
}

func (s *Service) GetRecipient(ctx context.Context, id int) (models.WebexRecipient, error) {
	return s.Recipients.Get(ctx, id)
}

func (s *Service) EnsurePersonRecipientByEmail(ctx context.Context, email string) (models.WebexRecipient, error) {
	recips, err := s.Recipients.ListByEmail(ctx, email)
	if err != nil {
		return models.WebexRecipient{}, fmt.Errorf("listing recipients by email: %w", err)
	}

	if len(recips) == 1 {
		return recips[0], nil
	}

	if len(recips) > 1 {
		return getMostActive(recips), nil
	}

	wxr, err := s.WebexClient.ListPeople(email)
	if err != nil {
		return models.WebexRecipient{}, fmt.Errorf("fetching people from webex api: %w", err)
	}

	if len(wxr) == 0 {
		return models.WebexRecipient{}, fmt.Errorf("no webex recipients found for email %s", email)
	}

	r := getMostActive(peopleToRecipients(wxr))
	r, err = s.Recipients.Upsert(ctx, r)
	if err != nil {
		return models.WebexRecipient{}, fmt.Errorf("upserting webex recipient: %w", err)
	}

	return r, nil
}

func getMostActive(recips []models.WebexRecipient) models.WebexRecipient {
	if len(recips) > 1 {
		sort.Slice(recips, func(i, j int) bool {
			return recips[i].LastActivity.After(recips[j].LastActivity)
		})
	}
	return recips[0]
}

func peopleToRecipients(webexPpl []webex.Person) []models.WebexRecipient {
	var toUpsert []models.WebexRecipient
	for _, p := range webexPpl {
		if len(p.Emails) == 0 {
			continue
		}

		r := models.WebexRecipient{
			WebexID:      p.ID,
			Name:         p.DisplayName,
			Email:        &p.Emails[0],
			Type:         "person",
			LastActivity: p.LastActivity,
		}

		toUpsert = append(toUpsert, r)
	}

	return toUpsert
}
