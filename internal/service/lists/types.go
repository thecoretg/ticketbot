package lists

import (
	"context"
	"fmt"

	"github.com/thecoretg/ticketbot/models"
)

// itemLabel is what the synced ConnectWise data says about one item.
type itemLabel struct {
	Label  string
	Detail string // secondary context; see models.ListItemTypeInfo.DetailLabel
}

// itemType is the per-type behaviour: how to check one item exists (returning its label) and how
// to label many at once. Register a new type here and in models.ListItemTypes.
type itemType struct {
	get    func(ctx context.Context, id int) (itemLabel, error)
	labels func(ctx context.Context, ids []int) (map[int]itemLabel, error)
}

func (s *Service) itemTypes() map[models.ListItemType]itemType {
	return map[models.ListItemType]itemType{
		models.ListItemContact: {
			get: func(ctx context.Context, id int) (itemLabel, error) {
				c, err := s.Contacts.Get(ctx, id)
				if err != nil {
					return itemLabel{}, err
				}
				labels, err := s.contactLabels(ctx, []*models.Contact{c})
				if err != nil {
					return itemLabel{}, err
				}
				return labels[c.ID], nil
			},
			labels: func(ctx context.Context, ids []int) (map[int]itemLabel, error) {
				cs, err := s.Contacts.Search(ctx, models.ContactSearch{IDs: ids, Limit: len(ids)})
				if err != nil {
					return nil, err
				}
				return s.contactLabels(ctx, cs)
			},
		},
		models.ListItemCompany: {
			get: func(ctx context.Context, id int) (itemLabel, error) {
				c, err := s.Companies.Get(ctx, id)
				if err != nil {
					return itemLabel{}, err
				}
				return itemLabel{Label: c.Name}, nil
			},
			labels: func(ctx context.Context, ids []int) (map[int]itemLabel, error) {
				cs, err := s.Companies.Search(ctx, models.CompanySearch{IDs: ids, Limit: len(ids)})
				if err != nil {
					return nil, err
				}
				out := make(map[int]itemLabel, len(cs))
				for _, c := range cs {
					out[c.ID] = itemLabel{Label: c.Name}
				}
				return out, nil
			},
		},
	}
}

// contactLabels names each contact and, as detail, its company.
func (s *Service) contactLabels(ctx context.Context, cs []*models.Contact) (map[int]itemLabel, error) {
	companyIDs := make([]int, 0, len(cs))
	seen := map[int]bool{}
	for _, c := range cs {
		if c.CompanyID != nil && !seen[*c.CompanyID] {
			seen[*c.CompanyID] = true
			companyIDs = append(companyIDs, *c.CompanyID)
		}
	}

	companies := map[int]string{}
	if len(companyIDs) > 0 {
		found, err := s.Companies.Search(ctx, models.CompanySearch{IDs: companyIDs, Limit: len(companyIDs)})
		if err != nil {
			return nil, fmt.Errorf("resolving company names: %w", err)
		}
		for _, co := range found {
			companies[co.ID] = co.Name
		}
	}

	out := make(map[int]itemLabel, len(cs))
	for _, c := range cs {
		l := itemLabel{Label: contactLabel(c)}
		if c.CompanyID != nil {
			l.Detail = companies[*c.CompanyID]
		}
		out[c.ID] = l
	}
	return out, nil
}

// labelItems fills in Label, Detail and Missing for items of one type.
func (s *Service) labelItems(ctx context.Context, t models.ListItemType, items []*models.ListItem) ([]models.ListItem, error) {
	out := make([]models.ListItem, 0, len(items))
	if len(items) == 0 {
		return out, nil
	}

	info, _ := t.Info()
	labels := map[int]itemLabel{}
	if it, ok := s.itemTypes()[t]; ok {
		ids := make([]int, 0, len(items))
		for _, i := range items {
			ids = append(ids, i.ItemID)
		}
		var err error
		if labels, err = it.labels(ctx, ids); err != nil {
			return nil, fmt.Errorf("resolving %s names: %w", t, err)
		}
	}

	for _, i := range items {
		li := *i
		if l, ok := labels[i.ItemID]; ok {
			li.Label, li.Detail = l.Label, l.Detail
		} else {
			li.Label = fmt.Sprintf("%s #%d", info.Label, i.ItemID)
			li.Missing = true
		}
		out = append(out, li)
	}

	return out, nil
}

// contactLabel matches the dashboard's wfLookupLabel: "First Last", falling back to the id.
func contactLabel(c *models.Contact) string {
	name := c.FirstName
	if c.LastName != nil && *c.LastName != "" {
		if name != "" {
			name += " "
		}
		name += *c.LastName
	}
	if name == "" {
		return fmt.Sprintf("Contact %d", c.ID)
	}
	return name
}
