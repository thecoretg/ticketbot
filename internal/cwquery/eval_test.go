package cwquery

import (
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

func testDoc() map[string]any {
	return map[string]any{
		"id":           float64(1234),
		"summary":      "Printer HELP needed",
		"board":        map[string]any{"id": float64(7), "name": "Dallas Help Desk"},
		"status":       map[string]any{"id": float64(16), "name": "New"},
		"company":      map[string]any{"id": float64(250), "identifier": "ACME", "name": "Acme Corp"},
		"priority":     map[string]any{"id": float64(4), "name": "Priority 3 - Normal"},
		"closedFlag":   true,
		"budgetHours":  float64(2.5),
		"resources":    "jdoe, asmith",
		"closedDate":   "2026-03-01T12:00:00Z",
		"stringNumber": "42",
		"latestNote": map[string]any{
			"id":                   float64(99),
			"text":                 "Please reboot the printer",
			"internalAnalysisFlag": true,
			"member":               map[string]any{"identifier": "jdoe"},
		},
		"changed": map[string]any{"status": true},
	}
}

func TestEval(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		// equality / case-insensitivity
		{"", true},
		{"id = 1234", true},
		{"id = '1234'", true},
		{"id != 1234", false},
		{"summary = 'printer help needed'", true},
		{"Summary = 'Printer HELP needed'", true},
		{"STATUS/NAME = 'new'", true},
		{"status/name != 'Closed'", true},
		{"company/id = 250", true},
		{"company/identifier = 'acme'", true},
		{"stringNumber = 42", true},

		// contains / like
		{"summary contains 'help'", true},
		{"summary contains 'HELP'", true},
		{"summary contains 'fax'", false},
		{"summary not contains 'fax'", true},
		{"summary like 'printer*'", true},
		{"summary like 'printer%'", true},
		{"summary like '*needed'", true},
		{"summary like 'printer'", false},
		{"summary like 'Printer HELP need__'", true},
		{"summary like '*a.b*'", false},
		{"summary not like 'printer*'", false},
		{"latestNote/text contains 'reboot'", true},
		{"resources contains 'asmith'", true},

		// in
		{"status/name in ('New', 'Assigned')", true},
		{"status/name in ('Closed', 'Assigned')", false},
		{"status/name not in ('Closed')", true},
		{"company/id in (1, 2, 250)", true},
		{"priority/id in ('4')", true},

		// ordering
		{"budgetHours > 2", true},
		{"budgetHours >= 2.5", true},
		{"budgetHours < 2.5", false},
		{"budgetHours <= 2.5", true},
		{"id > 1000 and id < 2000", true},
		{"stringNumber > 41", true},
		{"status/name > 'm'", true},
		{"status/name < 'm'", false},
		{"closedDate > [2026-01-01T00:00:00Z]", true},
		{"closedDate < [2026-01-01]", false},
		{"closedDate = [2026-03-01T12:00:00Z]", true},
		{"closedDate >= [2026-03-01T12:00:00Z]", true},

		// booleans
		{"closedFlag = true", true},
		{"closedFlag = 'true'", true},
		{"closedFlag = false", false},
		{"closedFlag != false", true},
		{"latestNote/internalAnalysisFlag = true", true},
		{"changed/status = true", true},
		{"changed/summary = true", false},

		// null looseness (missing fields)
		{"owner = null", true},
		{"owner/id = null", true},
		{"owner/id = 0", true},
		{"owner/id = ''", true},
		{"owner/id = false", true},
		{"owner/id = 5", false},
		{"owner/id != null", false},
		{"owner/id != 5", true},
		{"owner/id > 0", false},
		{"owner/id < 0", false},
		{"owner/id in (null)", true},
		{"changed/owner = false", true},
		{"changed/owner = true", false},
		{"status = null", false}, // object present
		{"status != null", true},
		{"summary = null", false},

		// boolean logic + precedence
		{"status/name = 'New' and company/id = 250", true},
		{"status/name = 'Closed' or company/id = 250", true},
		{"status/name = 'Closed' or company/id = 1 or id = 1234", true},
		{"status/name = 'Closed' and company/id = 250 or id = 1234", true},
		{"status/name = 'Closed' and (company/id = 250 or id = 1234)", false},
		{"not status/name = 'Closed'", true},
		{"not (status/name = 'New' and closedFlag = true)", false},
		{"NOT status/name = 'New' OR id = 1234", true},

		// type mismatches are false, not errors
		{"summary > 5", false},
		{"budgetHours contains 'x'", false},
		{"status = 'New'", false},
		{"status contains 'New'", false},
		{"board/name/foo = 'x'", false},
		{"board/name/foo = null", true},
	}

	doc := testDoc()
	for _, c := range cases {
		q, err := Compile(c.src)
		if err != nil {
			t.Errorf("%q: compile: %v", c.src, err)
			continue
		}
		got, err := q.Eval(doc)
		if err != nil {
			t.Errorf("%q: eval: %v", c.src, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %v, want %v", c.src, got, c.want)
		}
	}
}

func TestEvalNoNote(t *testing.T) {
	doc := NewDocument(&psa.Ticket{ID: 1}, nil, Changes{})
	for src, want := range map[string]bool{
		"latestNote = null":                       true,
		"latestNote/text contains 'x'":            false,
		"latestNote/internalAnalysisFlag = true":  false,
		"latestNote/internalAnalysisFlag = false": true,
		"changed/status = true":                   false,
		"newNote = true":                          false,
		"old/status/name = 'New'":                 false,
	} {
		q, err := Compile(src)
		if err != nil {
			t.Fatal(err)
		}
		got, _ := q.Eval(doc)
		if got != want {
			t.Errorf("%q = %v, want %v", src, got, want)
		}
	}
}

func TestNewDocumentFromTicket(t *testing.T) {
	tk := &psa.Ticket{ID: 555, Summary: "VPN down", ClosedFlag: false}
	tk.Board.ID = 3
	tk.Board.Name = "Network"
	tk.Company.ID = 42
	tk.Company.Identifier = "ACME"
	tk.Status.ID = 9
	tk.Status.Name = "New"
	tk.Priority.ID = 2
	tk.Priority.Name = "Priority 1 - Emergency"
	tk.Info.UpdatedBy = "jdoe"

	note := &psa.ServiceTicketNote{ID: 10, Text: "Checked the firewall", InternalAnalysisFlag: true}
	note.Member.Identifier = "asmith"

	doc := NewDocument(tk, note, Changes{NewNote: true, Fields: []models.FieldChange{
		{Field: "status", Old: ticketdiff.Ref{ID: 8, Name: "Assigned"}, New: ticketdiff.Ref{ID: 9, Name: "New"}},
		{Field: "priority", Old: ticketdiff.Ref{ID: 1}, New: ticketdiff.Ref{ID: 2}},
		{Field: "resources", Old: []string{"asmith", "jdoe"}, New: []string{"jdoe"}},
	}})

	for src, want := range map[string]bool{
		"id = 555":               true,
		"summary contains 'vpn'": true,
		"board/id = 3 and board/name = 'network'":            true,
		"company/identifier = 'acme'":                        true,
		"priority/name like 'Priority 1*'":                   true,
		"closedFlag = false":                                 true, // omitted zero value -> null -> loose false
		"closedFlag = true":                                  false,
		"owner/id = null":                                    true,
		"_info/updatedBy = 'jdoe'":                           true,
		"latestNote/text contains 'firewall'":                true,
		"latestNote/internalAnalysisFlag = true":             true,
		"latestNote/member/identifier = 'asmith'":            true,
		"changed/status = true and changed/priority = true":  true,
		"changed/summary = true":                             false,
		"newNote = true":                                     true,
		"old/status/name = 'assigned' and old/status/id = 8": true,
		"old/priority/id = 1":                                true,
		"old/resources contains 'asmith'":                    true,
		"old/summary = null":                                 true,
	} {
		q, err := Compile(src)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		got, err := q.Eval(doc)
		if err != nil {
			t.Fatalf("%q: %v", src, err)
		}
		if got != want {
			t.Errorf("%q = %v, want %v", src, got, want)
		}
	}
}

func TestNewDocumentNilTicket(t *testing.T) {
	doc := NewDocument(nil, nil, Changes{})
	q, err := Compile("id = null and latestNote = null")
	if err != nil {
		t.Fatal(err)
	}
	got, err := q.Eval(doc)
	if err != nil || !got {
		t.Fatalf("got %v, %v; want true", got, err)
	}
}

func TestEvalInList(t *testing.T) {
	env := Env{Lists: Lists{
		3: NewListSet(250, 999),
		5: NewListSet("acme"),
		6: NewListSet("42"),
	}}
	cases := []struct {
		src  string
		want bool
	}{
		{"company/id in list 3", true},
		{"company/id not in list 3", false},
		{"company/id in list 4", false}, // unknown list is empty
		{"company/id not in list 4", true},
		{"company/identifier in list 5", true}, // strings compare case-insensitively
		{"company/identifier in list 6", false},
		{"stringNumber in list 6", true}, // numeric strings canonicalize to numbers
		{"id in list 3", false},
		{"contact/id in list 3", false}, // missing field never matches
		{"contact/id not in list 3", true},
		{"company/id in list 3 and status/name = 'New'", true},
		{"not company/id in list 3", false},
	}

	doc := testDoc()
	for _, c := range cases {
		q, err := Compile(c.src)
		if err != nil {
			t.Errorf("%q: compile: %v", c.src, err)
			continue
		}
		got, err := q.EvalEnv(doc, env)
		if err != nil {
			t.Errorf("%q: %v", c.src, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q = %v, want %v", c.src, got, c.want)
		}
	}

	// Plain Eval has no lists: membership is always false.
	for src, want := range map[string]bool{"company/id in list 3": false, "company/id not in list 3": true} {
		q, _ := Compile(src)
		if got, _ := q.Eval(doc); got != want {
			t.Errorf("Eval %q = %v, want %v", src, got, want)
		}
	}
}

func TestListSetKeys(t *testing.T) {
	s := NewListSet(123, "ACME", true)
	for _, v := range []any{float64(123), "123", " 123 ", int64(123), "acme", "Acme", true, "true"} {
		if !s.Has(v) {
			t.Errorf("Has(%#v) = false", v)
		}
	}
	for _, v := range []any{float64(124), "acme corp", nil, map[string]any{}, []any{}} {
		if s.Has(v) {
			t.Errorf("Has(%#v) = true", v)
		}
	}
	var none Lists
	if none.Has(1, 1) {
		t.Error("nil Lists.Has = true")
	}
}

func TestListRefs(t *testing.T) {
	e, err := Parse("(a in list 1 or b = 2) and not c not in list 7")
	if err != nil {
		t.Fatal(err)
	}
	refs := ListRefs(e)
	if len(refs) != 2 || refs[0].ListID != 1 || refs[0].Path[0] != "a" || refs[1].ListID != 7 || refs[1].Path[0] != "c" {
		t.Errorf("refs = %+v", refs)
	}
	if refs[0].Pos != 11 || refs[1].Pos != 45 {
		t.Errorf("positions = %d, %d", refs[0].Pos, refs[1].Pos)
	}

	e, _ = Parse("a = 1")
	if refs := ListRefs(e); len(refs) != 0 {
		t.Errorf("refs = %+v, want none", refs)
	}
}
