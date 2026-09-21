package msgtemplate

import (
	"strings"
	"testing"

	"github.com/thecoretg/ticketbot/models"
)

func ticket() *models.FullTicket {
	last := "Smith"
	content := "Line one\nLine two that is long"
	p := "Priority 2"
	return &models.FullTicket{
		Ticket:  models.Ticket{ID: 42, Summary: "Printer down", PriorityName: &p},
		Board:   models.Board{Name: "Help Desk"},
		Status:  models.TicketStatus{Name: "New"},
		Company: models.Company{Name: "Acme"},
		Contact: &models.Contact{FirstName: "Ann", LastName: &last},
		Owner:   &models.Member{FirstName: "Jane", LastName: "Doe"},
		Resources: []*models.Member{
			{FirstName: "Jane", LastName: "Doe"},
			{FirstName: "Bob", LastName: "Ray"},
		},
		LatestNote: &models.FullTicketNote{
			TicketNote: models.TicketNote{Content: &content},
			Member:     &models.Member{FirstName: "Bob", LastName: "Ray"},
		},
	}
}

func TestRender(t *testing.T) {
	ctx := Context{Ticket: ticket(), StepTitle: "Escalate", IsNew: true, CompanyID: "acme", MaxNoteLen: 12}
	got := Render("{{event}} {{ ticket.link }} {{ticket.summary}} | {{board}}/{{status}}/{{priority}} | {{company}} {{contact}} {{owner}} [{{resources}}] by {{rule}}", ctx)
	want := "New Ticket [42](https://na.myconnectwise.net/v4_6_release/services/system_io/Service/fv_sr100_request.rails?service_recid=42&companyName=acme) Printer down | Help Desk/New/Priority 2 | Acme Ann Smith Jane Doe [Jane Doe, Bob Ray] by Escalate"
	if got != want {
		t.Errorf("got  %q\nwant %q", got, want)
	}

	note := Render("{{note.author}}: {{note.text}}\n{{note.quote}}", ctx)
	if !strings.HasPrefix(note, "Bob Ray: Line one\nLin...\n**Latest Note Sent By:** Bob Ray\n> Line one\n> Lin...") {
		t.Errorf("note render = %q", note)
	}

	if got := Render("{{event}}", Context{Ticket: ticket()}); got != "Ticket Updated" {
		t.Errorf("event for update = %q", got)
	}
	chg := Context{Ticket: ticket(), Changes: []models.FieldChange{{Field: "status", Old: "New", New: "Assigned"}}}
	if got := Render("{{changes}}", chg); got != "Status: New → Assigned" {
		t.Errorf("changes = %q", got)
	}
	if got := Render("x {{unknown}} y", ctx); got != "x  y" {
		t.Errorf("unknown placeholder should render empty: %q", got)
	}
	if got := Render("{{owner}}|{{contact}}|{{note.text}}", Context{Ticket: &models.FullTicket{}}); got != "||" {
		t.Errorf("nil parts should render empty: %q", got)
	}
}

func TestValidate(t *testing.T) {
	if err := Validate("plain text {{ticket.id}} {{ note.quote }}"); err != nil {
		t.Error(err)
	}
	if err := Validate("{{ticket.id}} {{bogus}}"); err == nil || !strings.Contains(err.Error(), "{{bogus}}") {
		t.Errorf("expected unknown placeholder error, got %v", err)
	}
	for _, p := range Placeholders {
		if err := Validate("{{" + p.Name + "}}"); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
}
