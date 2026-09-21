// Package lists manages admin-defined item lists that workflow conditions reference as
// `path in list <id>`.
package lists

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Service struct {
	Lists     repos.ListRepository
	Companies repos.CompanyRepository
	Contacts  repos.ContactRepository
	// Workflows is scanned for rules that reference a list; the repo is used directly (not the
	// workflow service) so the two services do not import each other.
	Workflows repos.WorkflowRepository
}

type Params struct {
	Lists     repos.ListRepository
	Companies repos.CompanyRepository
	Contacts  repos.ContactRepository
	Workflows repos.WorkflowRepository
}

func New(p Params) *Service {
	return &Service{Lists: p.Lists, Companies: p.Companies, Contacts: p.Contacts, Workflows: p.Workflows}
}

// Types returns the registered item types.
func (s *Service) Types() []models.ListItemTypeInfo {
	return models.ListItemTypes
}

func (s *Service) List(ctx context.Context) ([]*models.List, error) {
	return s.Lists.List(ctx)
}

// Get returns the list with its members (labelled) and the workflow rules that use it.
func (s *Service) Get(ctx context.Context, id int) (*models.ListDetail, error) {
	l, err := s.Lists.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	items, err := s.Lists.Items(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("loading items: %w", err)
	}
	labelled, err := s.labelItems(ctx, l.ItemType, items)
	if err != nil {
		return nil, err
	}

	refs, err := s.References(ctx, id)
	if err != nil {
		return nil, err
	}

	return &models.ListDetail{List: *l, Items: labelled, UsedBy: refs}, nil
}

func (s *Service) Create(ctx context.Context, l *models.List) (*models.List, error) {
	l.Name = strings.TrimSpace(l.Name)
	l.Description = strings.TrimSpace(l.Description)
	if l.Name == "" {
		return nil, &models.ListValidationError{Field: "name", Message: "name is required"}
	}
	if !l.ItemType.Valid() {
		return nil, &models.ListValidationError{Field: "item_type", Message: fmt.Sprintf("item_type must be one of %s (got %q)", typeNames(), l.ItemType)}
	}

	return s.Lists.Insert(ctx, l)
}

// Update renames or re-describes a list. The item type is fixed for the life of the list.
func (s *Service) Update(ctx context.Context, id int, name, description string) (*models.List, error) {
	current, err := s.Lists.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	name = strings.TrimSpace(name)
	if name == "" {
		return nil, &models.ListValidationError{Field: "name", Message: "name is required"}
	}
	current.Name = name
	current.Description = strings.TrimSpace(description)

	return s.Lists.Update(ctx, current)
}

// Delete removes a list unless a workflow rule still references it.
func (s *Service) Delete(ctx context.Context, id int) error {
	if _, err := s.Lists.Get(ctx, id); err != nil {
		return err
	}

	refs, err := s.References(ctx, id)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		return &models.ListInUseError{Refs: refs}
	}

	return s.Lists.Delete(ctx, id)
}

// AddItem adds an item after checking it exists in the synced ConnectWise data for the list's type.
func (s *Service) AddItem(ctx context.Context, listID, itemID int) (*models.ListItem, error) {
	l, err := s.Lists.Get(ctx, listID)
	if err != nil {
		return nil, err
	}

	it, ok := s.itemTypes()[l.ItemType]
	if !ok {
		return nil, fmt.Errorf("list %d has unregistered item type %q", listID, l.ItemType)
	}
	label, err := it.get(ctx, itemID)
	if err != nil {
		return nil, err
	}

	if _, err := s.Lists.AddItem(ctx, listID, itemID); err != nil {
		return nil, err
	}

	return &models.ListItem{ListID: listID, ItemID: itemID, Label: label.Label, Detail: label.Detail, AddedOn: time.Now().UTC()}, nil
}

func (s *Service) RemoveItem(ctx context.Context, listID, itemID int) error {
	if _, err := s.Lists.Get(ctx, listID); err != nil {
		return err
	}
	return s.Lists.RemoveItem(ctx, listID, itemID)
}

// References finds every workflow node whose condition mentions list id. Disabled nodes and
// workflows count; conditions that fail to parse are skipped.
func (s *Service) References(ctx context.Context, listID int) ([]models.ListReference, error) {
	wfs, err := s.Workflows.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading workflows: %w", err)
	}

	refs := []models.ListReference{}
	for _, w := range wfs {
		for _, n := range w.Nodes {
			if n.Kind != models.NodeIf {
				continue
			}
			expr, err := cwquery.Parse(n.Condition)
			if err != nil {
				continue
			}
			for _, ref := range cwquery.ListRefs(expr) {
				if ref.ListID == listID {
					refs = append(refs, models.ListReference{WorkflowID: w.ID, WorkflowName: w.Name, BoardName: w.BoardName, NodeID: n.ID, NodeTitle: n.Title})
					break
				}
			}
		}
	}

	return refs, nil
}

// Memberships loads every list's members in the shape the condition evaluator consumes. It
// satisfies workflow.ListLoader.
func (s *Service) Memberships(ctx context.Context) (cwquery.Lists, error) {
	ms, err := s.Lists.AllMemberships(ctx)
	if err != nil {
		return nil, fmt.Errorf("loading list memberships: %w", err)
	}

	out := cwquery.Lists{}
	for _, m := range ms {
		set, ok := out[m.ListID]
		if !ok {
			set = cwquery.ListSet{}
			out[m.ListID] = set
		}
		set.Add(m.ItemID)
	}

	return out, nil
}

func typeNames() string {
	names := make([]string, 0, len(models.ListItemTypes))
	for _, t := range models.ListItemTypes {
		names = append(names, string(t.Type))
	}
	return strings.Join(names, ", ")
}
