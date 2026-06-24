package cwsvc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// PickerOption is a combobox entry for the admin panel: Label is the only text
// shown, Value is the underlying CW reference stored in a rule (id or identifier),
// and Hint is optional muted disambiguation text shown only in the dropdown.
type PickerOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Hint  string `json:"hint,omitempty"`
}

const pickerPageSize = "50"

// sanitizeCond strips characters that would break a ConnectWise condition string
// (single quotes delimit string literals in the CW query language).
func sanitizeCond(s string) string {
	return strings.ReplaceAll(strings.TrimSpace(s), "'", "")
}

// SearchCompanies live-searches CW companies by name. Value is the company
// identifier (which the ticket_update company field sets).
func (s *Service) SearchCompanies(ctx context.Context, q string) ([]PickerOption, error) {
	params := map[string]string{"pageSize": pickerPageSize, "orderBy": "name asc"}
	if q = sanitizeCond(q); q != "" {
		params["conditions"] = fmt.Sprintf("name like '%%%s%%'", q)
	}

	companies, err := s.CWClient.ListCompanies(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("listing companies: %w", err)
	}

	out := make([]PickerOption, 0, len(companies))
	for _, c := range companies {
		out = append(out, PickerOption{Label: c.Name, Value: c.Identifier, Hint: c.Identifier})
	}
	return out, nil
}

// SearchContacts live-searches CW contacts within a company (by identifier),
// optionally filtered by name. Value is the contact id.
func (s *Service) SearchContacts(ctx context.Context, companyIdentifier, q string) ([]PickerOption, error) {
	companyIdentifier = sanitizeCond(companyIdentifier)
	if companyIdentifier == "" {
		return nil, fmt.Errorf("company identifier is required")
	}

	cond := fmt.Sprintf("company/identifier='%s' and inactiveFlag=false", companyIdentifier)
	if q = sanitizeCond(q); q != "" {
		cond += fmt.Sprintf(" and (firstName like '%%%s%%' or lastName like '%%%s%%')", q, q)
	}
	params := map[string]string{"conditions": cond, "pageSize": pickerPageSize, "orderBy": "firstName asc"}

	contacts, err := s.CWClient.ListContacts(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("listing contacts: %w", err)
	}

	out := make([]PickerOption, 0, len(contacts))
	for _, c := range contacts {
		name := strings.TrimSpace(c.FirstName + " " + c.LastName)
		if name == "" {
			name = fmt.Sprintf("contact %d", c.ID)
		}
		out = append(out, PickerOption{Label: name, Value: strconv.Itoa(c.ID)})
	}
	return out, nil
}

// ListBoardTypes returns a board's active ticket types from the local sync as
// picker options (q filters by name client-side). Value is the type id; Label is
// the name a condition matches against.
func (s *Service) ListBoardTypes(ctx context.Context, boardID int, q string) ([]PickerOption, error) {
	types, err := s.Types.ListByBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("listing board types: %w", err)
	}

	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]PickerOption, 0, len(types))
	for _, t := range types {
		if t.Inactive || t.Deleted {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(t.Name), q) {
			continue
		}
		out = append(out, PickerOption{Label: t.Name, Value: strconv.Itoa(t.ID)})
	}
	return out, nil
}

// ListBoardSubTypes returns a board's active ticket subtypes from the local sync as
// picker options (q filters by name client-side). Value is the subtype id; Label is
// the name a condition matches against.
func (s *Service) ListBoardSubTypes(ctx context.Context, boardID int, q string) ([]PickerOption, error) {
	subtypes, err := s.SubTypes.ListByBoard(ctx, boardID)
	if err != nil {
		return nil, fmt.Errorf("listing board subtypes: %w", err)
	}

	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]PickerOption, 0, len(subtypes))
	for _, st := range subtypes {
		if st.Inactive || st.Deleted {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(st.Name), q) {
			continue
		}
		out = append(out, PickerOption{Label: st.Name, Value: strconv.Itoa(st.ID)})
	}
	return out, nil
}

// LiveBoardStatuses live-fetches the active statuses for a board (q filters by
// name client-side, since a board has few statuses). Value is the status id.
func (s *Service) LiveBoardStatuses(ctx context.Context, boardID int, q string) ([]PickerOption, error) {
	params := map[string]string{"pageSize": "100", "orderBy": "sortOrder asc", "conditions": "inactive=false"}
	statuses, err := s.CWClient.ListBoardStatuses(ctx, params, boardID)
	if err != nil {
		return nil, fmt.Errorf("listing board statuses: %w", err)
	}

	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]PickerOption, 0, len(statuses))
	for _, st := range statuses {
		if q != "" && !strings.Contains(strings.ToLower(st.Name), q) {
			continue
		}
		out = append(out, PickerOption{Label: st.Name, Value: strconv.Itoa(st.ID)})
	}
	return out, nil
}
