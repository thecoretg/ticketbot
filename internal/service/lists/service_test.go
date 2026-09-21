package lists

import (
	"context"
	"errors"
	"testing"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// fakeListRepo satisfies repos.ListRepository via embedding; only what the service calls is real.
type fakeListRepo struct {
	repos.ListRepository
	lists   map[int]*models.List
	items   map[int][]int
	deleted []int
}

func (f *fakeListRepo) Get(_ context.Context, id int) (*models.List, error) {
	l, ok := f.lists[id]
	if !ok {
		return nil, models.ErrListNotFound
	}
	return l, nil
}

func (f *fakeListRepo) Insert(_ context.Context, l *models.List) (*models.List, error) {
	l.ID = len(f.lists) + 1
	f.lists[l.ID] = l
	return l, nil
}

func (f *fakeListRepo) Delete(_ context.Context, id int) error {
	delete(f.lists, id)
	f.deleted = append(f.deleted, id)
	return nil
}

func (f *fakeListRepo) Items(_ context.Context, id int) ([]*models.ListItem, error) {
	var out []*models.ListItem
	for _, it := range f.items[id] {
		out = append(out, &models.ListItem{ListID: id, ItemID: it})
	}
	return out, nil
}

func (f *fakeListRepo) AddItem(_ context.Context, listID, itemID int) (bool, error) {
	f.items[listID] = append(f.items[listID], itemID)
	return true, nil
}

func (f *fakeListRepo) AllMemberships(_ context.Context) ([]models.ListMembership, error) {
	var out []models.ListMembership
	for id, items := range f.items {
		for _, it := range items {
			out = append(out, models.ListMembership{ListID: id, ItemID: it})
		}
	}
	return out, nil
}

type fakeContactRepo struct {
	repos.ContactRepository
	contacts map[int]*models.Contact
}

func (f *fakeContactRepo) Get(_ context.Context, id int) (*models.Contact, error) {
	c, ok := f.contacts[id]
	if !ok {
		return nil, models.ErrContactNotFound
	}
	return c, nil
}

func (f *fakeContactRepo) Search(_ context.Context, s models.ContactSearch) ([]*models.Contact, error) {
	var out []*models.Contact
	for _, id := range s.IDs {
		if c, ok := f.contacts[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

type fakeCompanyRepo struct {
	repos.CompanyRepository
	companies map[int]*models.Company
}

func (f *fakeCompanyRepo) Get(_ context.Context, id int) (*models.Company, error) {
	c, ok := f.companies[id]
	if !ok {
		return nil, models.ErrCompanyNotFound
	}
	return c, nil
}

func (f *fakeCompanyRepo) Search(_ context.Context, s models.CompanySearch) ([]*models.Company, error) {
	var out []*models.Company
	for _, id := range s.IDs {
		if c, ok := f.companies[id]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

type fakeWorkflowRepo struct {
	repos.WorkflowRepository
	wfs []*models.Workflow
}

func (f *fakeWorkflowRepo) List(_ context.Context) ([]*models.Workflow, error) {
	return f.wfs, nil
}

func str(s string) *string { return &s }
func num(n int) *int       { return &n }

func newTestService() (*Service, *fakeListRepo) {
	lr := &fakeListRepo{
		lists: map[int]*models.List{
			7: {ID: 7, Name: "Drop", ItemType: models.ListItemContact},
			8: {ID: 8, Name: "VIP Companies", ItemType: models.ListItemCompany},
		},
		items: map[int][]int{7: {123, 999}},
	}
	svc := New(Params{
		Lists:     lr,
		Contacts:  &fakeContactRepo{contacts: map[int]*models.Contact{123: {ID: 123, FirstName: "Jane", LastName: str("Doe"), CompanyID: num(50)}}},
		Companies: &fakeCompanyRepo{companies: map[int]*models.Company{50: {ID: 50, Name: "Acme"}}},
		Workflows: &fakeWorkflowRepo{wfs: []*models.Workflow{
			{ID: 1, Name: "Help Desk", BoardName: "HD", Nodes: []models.Node{
				{ID: "r1", Kind: models.NodeIf, Title: "drop", Condition: "contact/id in list 7"},
				{ID: "r2", Kind: models.NodeIf, Title: "other", Condition: "status/name = 'New'"},
				{ID: "r3", Kind: models.NodeIf, Title: "broken", Condition: "summary ="},
				{ID: "t", Kind: models.NodeTrigger, Title: "trigger", Condition: "contact/id in list 7"}, // not an if: ignored
			}},
			{ID: 2, Name: "Sales", Nodes: []models.Node{{ID: "r4", Kind: models.NodeIf, Title: "vip", Condition: "company/id not in list 8 and id > 1"}}},
		}},
	})
	return svc, lr
}

func TestCreateValidation(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()

	var ve *models.ListValidationError
	if _, err := svc.Create(ctx, &models.List{Name: "  ", ItemType: models.ListItemContact}); !errors.As(err, &ve) || ve.Field != "name" {
		t.Errorf("blank name err = %v", err)
	}
	if _, err := svc.Create(ctx, &models.List{Name: "x", ItemType: "ticket"}); !errors.As(err, &ve) || ve.Field != "item_type" {
		t.Errorf("bad type err = %v", err)
	}
	l, err := svc.Create(ctx, &models.List{Name: "  New  ", ItemType: models.ListItemCompany})
	if err != nil || l.Name != "New" || l.ID == 0 {
		t.Errorf("create = %+v, %v", l, err)
	}
}

func TestAddItemChecksExistence(t *testing.T) {
	svc, lr := newTestService()
	ctx := context.Background()

	if _, err := svc.AddItem(ctx, 7, 456); !errors.Is(err, models.ErrContactNotFound) {
		t.Errorf("unknown contact err = %v", err)
	}
	it, err := svc.AddItem(ctx, 7, 123)
	if err != nil || it.Label != "Jane Doe" || it.Detail != "Acme" {
		t.Errorf("add = %+v, %v", it, err)
	}
	if _, err := svc.AddItem(ctx, 8, 51); !errors.Is(err, models.ErrCompanyNotFound) {
		t.Errorf("unknown company err = %v", err)
	}
	if _, err := svc.AddItem(ctx, 99, 1); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("unknown list err = %v", err)
	}
	if len(lr.items[7]) != 3 {
		t.Errorf("items = %v", lr.items[7])
	}
}

func TestGetLabelsAndReferences(t *testing.T) {
	svc, _ := newTestService()
	d, err := svc.Get(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Items) != 2 || d.Items[0].Label != "Jane Doe" || d.Items[0].Detail != "Acme" || d.Items[0].Missing {
		t.Errorf("items[0] = %+v", d.Items[0])
	}
	if d.Items[1].Label != "Contact #999" || !d.Items[1].Missing {
		t.Errorf("items[1] = %+v", d.Items[1])
	}
	if len(d.UsedBy) != 1 || d.UsedBy[0].WorkflowName != "Help Desk" || d.UsedBy[0].NodeTitle != "drop" {
		t.Errorf("used_by = %+v", d.UsedBy)
	}
}

func TestDeleteBlockedWhenReferenced(t *testing.T) {
	svc, lr := newTestService()
	ctx := context.Background()

	var inUse *models.ListInUseError
	if err := svc.Delete(ctx, 8); !errors.As(err, &inUse) || len(inUse.Refs) != 1 || inUse.Refs[0].NodeID != "r4" {
		t.Errorf("delete referenced err = %v", err)
	}
	if len(lr.deleted) != 0 {
		t.Errorf("referenced list was deleted")
	}

	lr.lists[9] = &models.List{ID: 9, Name: "Unused", ItemType: models.ListItemContact}
	if err := svc.Delete(ctx, 9); err != nil || len(lr.deleted) != 1 {
		t.Errorf("delete unreferenced = %v, deleted %v", err, lr.deleted)
	}
	if err := svc.Delete(ctx, 99); !errors.Is(err, models.ErrListNotFound) {
		t.Errorf("delete unknown err = %v", err)
	}
}

func TestMemberships(t *testing.T) {
	svc, _ := newTestService()
	ls, err := svc.Memberships(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !ls.Has(7, float64(123)) || !ls.Has(7, "999") || ls.Has(7, float64(1)) || ls.Has(8, float64(123)) {
		t.Errorf("memberships = %+v", ls)
	}
}
