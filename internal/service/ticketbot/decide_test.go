package ticketbot

import (
	"encoding/json"
	"testing"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

func cwTicket() *psa.Ticket {
	t := &psa.Ticket{ID: 1, Summary: "Printer down", Resources: "jdoe"}
	t.Board.ID = 1
	t.Status.ID, t.Status.Name = 10, "New"
	t.Company.ID = 100
	t.Priority.ID, t.Priority.Name = 3, "P3"
	return t
}

func stored(raw bool, latestNote *int) *models.Ticket {
	rsc := "jdoe"
	s := &models.Ticket{ID: 1, Summary: "Printer down", BoardID: 1, StatusID: 10, CompanyID: 100, Resources: &rsc, LatestNoteID: latestNote}
	if raw {
		s.Raw, _ = json.Marshal(cwTicket())
	}
	return s
}

func note(id int) *psa.ServiceTicketNote {
	return &psa.ServiceTicketNote{ID: id, Text: "hello"}
}

func TestDecide(t *testing.T) {
	n5 := 5

	cases := []struct {
		name    string
		stored  *models.Ticket
		f       *cwsvc.Fetched
		isNew   bool
		changed []string
		newNote bool
	}{
		{"never seen", nil, &cwsvc.Fetched{Ticket: cwTicket(), Note: note(5)}, true, nil, true},
		{"never seen no note", nil, &cwsvc.Fetched{Ticket: cwTicket()}, true, nil, false},
		{"identical", stored(true, &n5), &cwsvc.Fetched{Ticket: cwTicket(), Note: note(5)}, false, nil, false},
		{"new note only", stored(true, &n5), &cwsvc.Fetched{Ticket: cwTicket(), Note: note(6)}, false, nil, true},
		{"first note", stored(true, nil), &cwsvc.Fetched{Ticket: cwTicket(), Note: note(6)}, false, nil, true},
		{"status change", stored(true, &n5), &cwsvc.Fetched{Ticket: func() *psa.Ticket { c := cwTicket(); c.Status.ID = 11; return c }(), Note: note(5)}, false, []string{ticketdiff.FieldStatus}, false},
		{"priority change with raw", stored(true, &n5), &cwsvc.Fetched{Ticket: func() *psa.Ticket { c := cwTicket(); c.Priority.ID = 1; return c }(), Note: note(5)}, false, []string{ticketdiff.FieldPriority}, false},
		// legacy rows (no raw) cannot see priority, only legacy fields
		{"priority change legacy", stored(false, &n5), &cwsvc.Fetched{Ticket: func() *psa.Ticket { c := cwTicket(); c.Priority.ID = 1; return c }(), Note: note(5)}, false, nil, false},
		{"summary change legacy", stored(false, &n5), &cwsvc.Fetched{Ticket: func() *psa.Ticket { c := cwTicket(); c.Summary = "x"; return c }(), Note: note(5)}, false, []string{ticketdiff.FieldSummary}, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := decide(c.stored, c.f)
			if d.IsNew != c.isNew {
				t.Errorf("IsNew = %v, want %v", d.IsNew, c.isNew)
			}
			if d.NewNote != c.newNote {
				t.Errorf("NewNote = %v, want %v", d.NewNote, c.newNote)
			}
			got := ticketdiff.FieldNames(d.Changes)
			if len(got) != len(c.changed) {
				t.Fatalf("changes = %v, want %v", got, c.changed)
			}
			for i := range got {
				if got[i] != c.changed[i] {
					t.Errorf("changes = %v, want %v", got, c.changed)
				}
			}
			wantChanged := c.isNew || c.newNote || len(c.changed) > 0
			if d.Changed() != wantChanged {
				t.Errorf("Changed() = %v, want %v", d.Changed(), wantChanged)
			}
		})
	}
}

func TestChangePayload(t *testing.T) {
	f := &cwsvc.Fetched{Ticket: cwTicket(), Note: note(9)}
	f.Ticket.Info.UpdatedBy = "jdoe"
	f.Note.Member.Identifier = "asmith"
	f.Note.Text = "   a long note body"

	p := changePayload(decision{NewNote: true}, f)
	if p.UpdatedBy != "jdoe" || p.NewNote == nil || p.NewNote.AuthorIdentifier != "asmith" || p.NewNote.Preview != "a long note body" {
		t.Errorf("unexpected payload %+v %+v", p, p.NewNote)
	}
	if p.Changes == nil {
		t.Error("Changes should be an empty slice, not nil")
	}

	if got := preview("abcdef", 3); got != "abc..." {
		t.Errorf("preview = %q", got)
	}
}
