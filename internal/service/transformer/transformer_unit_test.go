package transformer

import (
	"encoding/json"
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

func sampleTicket() *psa.Ticket {
	t := &psa.Ticket{ID: 42, Summary: "Printer broken"}
	t.Company.ID = 7
	t.Company.Identifier = "ACME"
	t.Company.Name = "Acme Corp"
	t.Board.ID = 3
	return t
}

func TestRenderSummaryTemplate(t *testing.T) {
	tk := sampleTicket()
	p := &UpdateSummaryParams{Summary: "[{{.Company.Identifier}}] {{.Summary}}"}
	if err := renderParams(p, tk); err != nil {
		t.Fatal(err)
	}
	if p.Summary != "[ACME] Printer broken" {
		t.Fatalf("got %q", p.Summary)
	}
}

func TestRenderMissingFieldErrors(t *testing.T) {
	tk := sampleTicket()
	p := &UpdateSummaryParams{Summary: "{{.Nonexistent}}"}
	if err := renderParams(p, tk); err == nil {
		t.Fatal("expected error for missing field")
	}
}

func TestValidateTemplatesCatchesSyntax(t *testing.T) {
	if err := validateTemplates(&UpdateSummaryParams{Summary: "{{.Summary"}); err == nil {
		t.Fatal("expected syntax error")
	}
	if err := validateTemplates(&UpdateSummaryParams{Summary: "{{.Summary}}"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRuleApplies(t *testing.T) {
	tk := sampleTicket() // board 3, summary "Printer broken", company "Acme Corp"/"ACME"
	board3 := 3
	board9 := 9
	cond := func(f, op, v string) []models.RuleCondition {
		return []models.RuleCondition{{Field: f, Operator: op, Value: v}}
	}
	cases := []struct {
		name   string
		rule   *models.TransformerRule
		isNew  bool
		expect bool
	}{
		{"all boards both", &models.TransformerRule{ApplyOn: "both"}, false, true},
		{"matching board", &models.TransformerRule{CwBoardID: &board3, ApplyOn: "both"}, true, true},
		{"other board", &models.TransformerRule{CwBoardID: &board9, ApplyOn: "both"}, true, false},
		{"new only on new", &models.TransformerRule{ApplyOn: "new"}, true, true},
		{"new only on update", &models.TransformerRule{ApplyOn: "new"}, false, false},
		{"updated only on update", &models.TransformerRule{ApplyOn: "updated"}, false, true},
		{"summary contains match", &models.TransformerRule{ApplyOn: "both", Conditions: cond("summary", "contains", "printer")}, true, true},
		{"summary contains miss", &models.TransformerRule{ApplyOn: "both", Conditions: cond("summary", "contains", "laptop")}, true, false},
		{"company equals match (ci)", &models.TransformerRule{ApplyOn: "both", Conditions: cond("company_identifier", "equals", "acme")}, true, true},
		{"company not_equals", &models.TransformerRule{ApplyOn: "both", Conditions: cond("company_name", "not_equals", "Acme Corp")}, true, false},
		{"starts_with match", &models.TransformerRule{ApplyOn: "both", Conditions: cond("summary", "starts_with", "Printer")}, true, true},
		{"two conditions all match", &models.TransformerRule{ApplyOn: "both", Conditions: []models.RuleCondition{
			{Field: "summary", Operator: "contains", Value: "broken"},
			{Field: "company_identifier", Operator: "equals", Value: "ACME"},
		}}, true, true},
		{"two conditions one miss", &models.TransformerRule{ApplyOn: "both", Conditions: []models.RuleCondition{
			{Field: "summary", Operator: "contains", Value: "broken"},
			{Field: "company_identifier", Operator: "equals", Value: "OTHER"},
		}}, true, false},
	}
	for _, c := range cases {
		if got := ruleApplies(c.rule, tk, c.isNew); got != c.expect {
			t.Errorf("%s: got %v want %v", c.name, got, c.expect)
		}
	}
}

func TestValidateConditions(t *testing.T) {
	if err := validateConditions([]models.RuleCondition{{Field: "summary", Operator: "contains", Value: "x"}}); err != nil {
		t.Fatalf("valid condition rejected: %v", err)
	}
	if err := validateConditions([]models.RuleCondition{{Field: "nonsense", Operator: "contains", Value: "x"}}); err == nil {
		t.Fatal("expected unknown-field error")
	}
	if err := validateConditions([]models.RuleCondition{{Field: "summary", Operator: "matches_regex", Value: "x"}}); err == nil {
		t.Fatal("expected unknown-operator error")
	}
}

func TestRegistryHasActions(t *testing.T) {
	reg := newRegistry()
	for _, a := range []string{"update_summary", "add_note"} {
		if _, ok := reg[a]; !ok {
			t.Errorf("missing action %q in registry", a)
		}
	}
}

func TestAddNoteParamsUnmarshal(t *testing.T) {
	var p AddNoteParams
	if err := json.Unmarshal([]byte(`{"text":"hi {{.Company.Name}}","internal":true}`), &p); err != nil {
		t.Fatal(err)
	}
	if !p.Internal || p.Text == "" {
		t.Fatalf("bad unmarshal: %+v", p)
	}
}
