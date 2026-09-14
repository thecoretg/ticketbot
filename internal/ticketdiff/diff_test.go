package ticketdiff

import (
	"slices"
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

func base() *psa.Ticket {
	t := &psa.Ticket{ID: 1, Summary: "Printer down", Resources: "jdoe, asmith", ClosedFlag: false}
	t.Board.ID, t.Board.Name = 1, "Help Desk"
	t.Status.ID, t.Status.Name = 10, "New"
	t.Owner.ID, t.Owner.Name = 5, "Jane Doe"
	t.Company.ID, t.Company.Name = 100, "Acme"
	t.Contact.ID, t.Contact.Name = 200, "Bob"
	t.Priority.ID, t.Priority.Name = 3, "P3"
	t.Type.ID, t.Type.Name = 7, "Hardware"
	t.SubType.ID, t.SubType.Name = 8, "Printer"
	t.Item.ID, t.Item.Name = 9, "Jam"
	return t
}

func fields(ch []models.FieldChange) []string {
	return FieldNames(ch)
}

func TestDiffNoChange(t *testing.T) {
	if ch := Diff(base(), base(), CuratedFields); len(ch) != 0 {
		t.Fatalf("expected no changes, got %+v", ch)
	}
}

func TestDiffNilOld(t *testing.T) {
	if ch := Diff(nil, base(), CuratedFields); ch != nil {
		t.Fatalf("expected nil, got %+v", ch)
	}
}

func TestDiffEachField(t *testing.T) {
	cases := []struct {
		field  string
		mutate func(*psa.Ticket)
	}{
		{FieldSummary, func(p *psa.Ticket) { p.Summary = "Printer up" }},
		{FieldStatus, func(p *psa.Ticket) { p.Status.ID = 11; p.Status.Name = "Closed" }},
		{FieldBoard, func(p *psa.Ticket) { p.Board.ID = 2 }},
		{FieldOwner, func(p *psa.Ticket) { p.Owner.ID = 0; p.Owner.Name = "" }},
		{FieldResources, func(p *psa.Ticket) { p.Resources = "jdoe" }},
		{FieldPriority, func(p *psa.Ticket) { p.Priority.ID = 1 }},
		{FieldType, func(p *psa.Ticket) { p.Type.ID = 70 }},
		{FieldSubType, func(p *psa.Ticket) { p.SubType.ID = 80 }},
		{FieldItem, func(p *psa.Ticket) { p.Item.ID = 90 }},
		{FieldCompany, func(p *psa.Ticket) { p.Company.ID = 101 }},
		{FieldContact, func(p *psa.Ticket) { p.Contact.ID = 201 }},
		{FieldClosedFlag, func(p *psa.Ticket) { p.ClosedFlag = true }},
	}

	for _, c := range cases {
		cur := base()
		c.mutate(cur)
		ch := Diff(base(), cur, CuratedFields)
		got := fields(ch)
		if !slices.Equal(got, []string{c.field}) {
			t.Errorf("%s: changed fields = %v, want [%s]", c.field, got, c.field)
		}
	}
}

func TestDiffStatusChangeShape(t *testing.T) {
	cur := base()
	cur.Status.ID, cur.Status.Name = 11, "Closed"
	ch := Diff(base(), cur, CuratedFields)
	if len(ch) != 1 {
		t.Fatalf("got %d changes", len(ch))
	}
	if ch[0].Old != (Ref{10, "New"}) || ch[0].New != (Ref{11, "Closed"}) {
		t.Errorf("unexpected old/new: %+v", ch[0])
	}
}

func TestDiffRefRenameIsNotChange(t *testing.T) {
	cur := base()
	cur.Status.Name = "Renamed status, same id"
	if ch := Diff(base(), cur, CuratedFields); len(ch) != 0 {
		t.Fatalf("rename with same id should not be a change: %+v", ch)
	}
}

func TestDiffResourcesOrderInsensitive(t *testing.T) {
	cur := base()
	cur.Resources = "asmith,jdoe"
	if ch := Diff(base(), cur, CuratedFields); len(ch) != 0 {
		t.Fatalf("reordered resources should not be a change: %+v", ch)
	}

	cur.Resources = ""
	ch := Diff(base(), cur, CuratedFields)
	if len(ch) != 1 || ch[0].Field != FieldResources {
		t.Fatalf("expected resources change, got %+v", ch)
	}
	if got := ch[0].New.([]string); len(got) != 0 {
		t.Errorf("empty resources should be an empty slice, got %v", got)
	}
}

func TestDiffRestrictedFields(t *testing.T) {
	cur := base()
	cur.Priority.ID = 1
	cur.Summary = "changed"
	got := fields(Diff(base(), cur, LegacyFields))
	if !slices.Equal(got, []string{FieldSummary}) {
		t.Errorf("legacy fields should ignore priority: %v", got)
	}
}

func TestFromStoredLegacyRoundTrip(t *testing.T) {
	owner, contact, rsc := 5, 200, "jdoe, asmith"
	stored := &models.Ticket{
		ID: 1, Summary: "Printer down", BoardID: 1, StatusID: 10, OwnerID: &owner,
		CompanyID: 100, ContactID: &contact, Resources: &rsc,
	}

	old := FromStored(stored)
	if ch := Diff(old, base(), LegacyFields); len(ch) != 0 {
		t.Fatalf("stored row should match ticket on legacy fields: %+v", ch)
	}

	cur := base()
	cur.Owner.ID = 6
	got := fields(Diff(old, cur, LegacyFields))
	if !slices.Equal(got, []string{FieldOwner}) {
		t.Errorf("got %v, want [owner]", got)
	}
}

func TestFromStoredNil(t *testing.T) {
	if FromStored(nil) != nil {
		t.Fatal("expected nil")
	}
}
