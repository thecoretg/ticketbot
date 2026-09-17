package cwsvc

import (
	"context"

	"github.com/thecoretg/ticketbot/models"
)

// SearchCompanies is the typeahead behind the condition builder's company picker.
func (s *Service) SearchCompanies(ctx context.Context, f models.CompanySearch) ([]*models.Company, error) {
	return s.Companies.Search(ctx, f)
}

// SearchContacts is the typeahead behind the condition builder's contact picker.
func (s *Service) SearchContacts(ctx context.Context, f models.ContactSearch) ([]*models.Contact, error) {
	return s.Contacts.Search(ctx, f)
}
