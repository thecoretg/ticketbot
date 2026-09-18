package ticketbot

import (
	"testing"

	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/models"
)

func TestLoopGuard(t *testing.T) {
	svc := &Service{Cfg: &models.Config{CWAPIMemberIdentifier: "TicketBot"}}

	botNote := note(9)
	botNote.Member.Identifier = "ticketbot"
	humanNote := note(9)
	humanNote.Member.Identifier = "jdoe"

	botTicket := cwTicket()
	botTicket.Info.UpdatedBy = "ticketbot"
	humanTicket := cwTicket()
	humanTicket.Info.UpdatedBy = "jdoe"

	cases := []struct {
		name   string
		f      *cwsvc.Fetched
		d      decision
		reason string
	}{
		{"bot note", &cwsvc.Fetched{Ticket: humanTicket, Note: botNote}, decision{NewNote: true}, "note_author"},
		{"human note", &cwsvc.Fetched{Ticket: botTicket, Note: humanNote}, decision{NewNote: true}, ""},
		{"bot field change", &cwsvc.Fetched{Ticket: botTicket}, decision{Changes: []models.FieldChange{{Field: "status"}}}, "updated_by"},
		{"human field change", &cwsvc.Fetched{Ticket: humanTicket}, decision{Changes: []models.FieldChange{{Field: "status"}}}, ""},
		{"new ticket by bot", &cwsvc.Fetched{Ticket: botTicket}, decision{IsNew: true}, ""},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := svc.loopGuard(c.f, c.d)
			got := ""
			if g != nil {
				got = g.Reason
			}
			if got != c.reason {
				t.Errorf("reason = %q, want %q", got, c.reason)
			}
		})
	}

	unset := &Service{Cfg: &models.Config{}}
	if g := unset.loopGuard(&cwsvc.Fetched{Ticket: botTicket}, decision{Changes: []models.FieldChange{{Field: "status"}}}); g != nil {
		t.Error("guard must be inert when no api member is configured")
	}
}
