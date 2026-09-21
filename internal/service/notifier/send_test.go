package notifier

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type fakeSender struct {
	repos.MessageSender
	sent []webex.Message
	err  error
}

func (f *fakeSender) PostMessage(_ context.Context, m *webex.Message) (*webex.Message, error) {
	if f.err != nil {
		return nil, f.err
	}
	f.sent = append(f.sent, *m)
	return m, nil
}

type fakeNotifRepo struct {
	repos.TicketNotificationRepository
	inserted []*models.TicketNotification
}

func (f *fakeNotifRepo) Insert(_ context.Context, n *models.TicketNotification) (*models.TicketNotification, error) {
	cp := *n
	cp.ID = len(f.inserted) + 1
	f.inserted = append(f.inserted, &cp)
	return &cp, nil
}

// emailRecipRepo extends fakeRecipRepo with ListByEmail for resources_owner resolution.
type emailRecipRepo struct {
	fakeRecipRepo
	byEmail map[string]*models.WebexRecipient
}

func (f *emailRecipRepo) ListByEmail(_ context.Context, email string) ([]*models.WebexRecipient, error) {
	if r, ok := f.byEmail[email]; ok {
		return []*models.WebexRecipient{r}, nil
	}
	return nil, nil
}

func (f *emailRecipRepo) Get(_ context.Context, id int) (*models.WebexRecipient, error) {
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return nil, models.ErrWebexRecipientNotFound
}

func namedRoom(id int, name string) *models.WebexRecipient {
	r := room(id)
	r.Name = name
	r.WebexID = "wx-" + name
	return r
}

func namedPerson(id int, email string) *models.WebexRecipient {
	r := person(id)
	r.Name = email
	r.Email = &email
	return r
}

type sendFixture struct {
	svc    *Service
	sender *fakeSender
	notifs *fakeNotifRepo
}

func newSendFixture(fwds map[int][]*models.NotifierForwardFull) *sendFixture {
	jane, bob, alice := namedPerson(10, "jane@x.com"), namedPerson(11, "bob@x.com"), namedPerson(12, "alice@x.com")
	recips := &emailRecipRepo{
		fakeRecipRepo: fakeRecipRepo{byID: map[int]*models.WebexRecipient{
			1: namedRoom(1, "Dallas"), 2: namedRoom(2, "Austin"), 10: jane, 11: bob, 12: alice,
		}},
		byEmail: map[string]*models.WebexRecipient{"jane@x.com": jane, "bob@x.com": bob, "alice@x.com": alice},
	}

	sender := &fakeSender{}
	notifs := &fakeNotifRepo{}
	return &sendFixture{
		sender: sender,
		notifs: notifs,
		svc: &Service{
			Cfg:           &models.Config{MaxMessageLength: 300},
			Forwards:      &fakeFwdRepo{bySource: fwds},
			WebexSvc:      webexsvc.New(nil, recips, nil),
			Notifications: notifs,
			MessageSender: sender,
			CWCompanyID:   "acme",
		},
	}
}

func fullTicket() *models.FullTicket {
	return &models.FullTicket{
		Ticket:  models.Ticket{ID: 42, Summary: "Printer down"},
		Company: models.Company{Name: "Acme"},
		Owner:   &models.Member{ID: 1, PrimaryEmail: "jane@x.com"},
		Resources: []*models.Member{
			{ID: 1, PrimaryEmail: "jane@x.com"},
			{ID: 2, PrimaryEmail: "bob@x.com"},
		},
		LatestNote: &models.FullTicketNote{
			TicketNote: models.TicketNote{ID: 7},
			Member:     &models.Member{ID: 2, PrimaryEmail: "bob@x.com"}, // bob wrote the note
		},
	}
}

func roomIntent(rule string, id int) workflow.NotifyIntent {
	return workflow.NotifyIntent{Step: workflow.StepRef{NodeID: rule, Title: rule}, Action: models.NotifyAction{Channel: models.ChannelWebexRoom, RecipientID: &id}}
}

func ownerIntent(rule string) workflow.NotifyIntent {
	return workflow.NotifyIntent{Step: workflow.StepRef{NodeID: rule, Title: rule}, Action: models.NotifyAction{Channel: models.ChannelResourcesOwner}}
}

func byRecipient(outs []Outcome) map[int]Outcome {
	m := make(map[int]Outcome)
	for _, o := range outs {
		if o.Recipient != nil {
			m[o.Recipient.ID] = o
		}
	}
	return m
}

func TestSendRoomAndResourcesOwner(t *testing.T) {
	fx := newSendFixture(nil)
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		IsNew:   true,
		Intents: []workflow.NotifyIntent{roomIntent("rooms", 1), ownerIntent("people")},
	})
	if err != nil {
		t.Fatal(err)
	}

	got := byRecipient(outs)
	// Dallas room + jane; bob authored the note and is excluded
	if len(got) != 2 || got[1].Result != ResultSent || got[10].Result != ResultSent {
		t.Fatalf("outcomes = %+v", outs)
	}
	if got[1].Step.NodeID != "rooms" || got[10].Step.NodeID != "people" {
		t.Errorf("attribution wrong: %+v", got)
	}
	if len(fx.sender.sent) != 2 || len(fx.notifs.inserted) != 2 {
		t.Errorf("sent=%d inserted=%d", len(fx.sender.sent), len(fx.notifs.inserted))
	}
	if got[10].Notification == nil || got[10].Notification.TicketNoteID == nil || *got[10].Notification.TicketNoteID != 7 {
		t.Errorf("notification row should reference the trigger note: %+v", got[10].Notification)
	}
}

func TestSendDryRun(t *testing.T) {
	fx := newSendFixture(nil)
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		DryRun:  true,
		Intents: []workflow.NotifyIntent{roomIntent("r", 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || outs[0].Result != ResultWouldSend {
		t.Fatalf("outcomes = %+v", outs)
	}
	if len(fx.sender.sent) != 0 || len(fx.notifs.inserted) != 0 {
		t.Error("dry run must not send or store")
	}
}

func TestSendDedupesRecipientsAcrossIntents(t *testing.T) {
	fx := newSendFixture(nil)
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		Intents: []workflow.NotifyIntent{roomIntent("first", 1), roomIntent("second", 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || outs[0].Step.NodeID != "first" {
		t.Fatalf("expected one send attributed to first rule: %+v", outs)
	}
}

func TestSendAppliesForwards(t *testing.T) {
	// jane forwards to alice and does not keep a copy
	fx := newSendFixture(map[int][]*models.NotifierForwardFull{10: {fwd(12, false, false)}})
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		Intents: []workflow.NotifyIntent{ownerIntent("people")},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := byRecipient(outs)
	if _, janeGotIt := got[10]; janeGotIt {
		t.Error("jane should have been forwarded away")
	}
	alice, ok := got[12]
	if !ok || alice.Result != ResultSent {
		t.Fatalf("alice should receive the forward: %+v", outs)
	}
	if len(alice.ForwardedFrom) != 1 || alice.ForwardedFrom[0] != "jane@x.com" {
		t.Errorf("forwarded_from = %v", alice.ForwardedFrom)
	}
	if alice.Step.NodeID != "people" {
		t.Errorf("forwarded recipient should inherit origin attribution: %+v", alice.Step)
	}
}

func TestSendResolveErrorIsPerIntent(t *testing.T) {
	fx := newSendFixture(nil)
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		Intents: []workflow.NotifyIntent{roomIntent("missing", 999), roomIntent("ok", 2)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 2 {
		t.Fatalf("outcomes = %+v", outs)
	}
	if outs[0].Result != ResultError || outs[0].Step.NodeID != "missing" || outs[0].Recipient != nil {
		t.Errorf("first outcome should be a resolve error: %+v", outs[0])
	}
	if outs[1].Result != ResultSent || outs[1].Recipient.ID != 2 {
		t.Errorf("second intent should still send: %+v", outs[1])
	}
}

func TestSendWebexFailureStillRecords(t *testing.T) {
	fx := newSendFixture(nil)
	fx.sender.err = errors.New("webex down")
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		Intents: []workflow.NotifyIntent{roomIntent("r", 1)},
	})
	if err != nil {
		t.Fatal(err)
	}
	if outs[0].Result != ResultError || outs[0].Err == nil {
		t.Errorf("expected error outcome: %+v", outs[0])
	}
	if len(fx.notifs.inserted) != 1 {
		t.Error("failed sends are still recorded to prevent retry storms")
	}
}

func TestSendNoIntents(t *testing.T) {
	fx := newSendFixture(nil)
	outs, err := fx.svc.Send(context.Background(), SendRequest{Ticket: fullTicket()})
	if err != nil || outs != nil {
		t.Errorf("got %v, %v", outs, err)
	}
	if _, err := fx.svc.Send(context.Background(), SendRequest{Intents: []workflow.NotifyIntent{roomIntent("r", 1)}}); err == nil {
		t.Error("nil ticket should error")
	}
}

func TestSendCustomMessageFollowsAttribution(t *testing.T) {
	fx := newSendFixture(nil)
	custom := roomIntent("custom", 1)
	custom.Action.Message = "{{event}}: {{ticket.summary}} ({{rule}})"
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		IsNew:   true,
		Intents: []workflow.NotifyIntent{custom, ownerIntent("default")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 2 || len(fx.sender.sent) != 2 {
		t.Fatalf("outcomes = %+v", outs)
	}

	var roomBody, personBody string
	for _, m := range fx.sender.sent {
		if m.RoomID != "" {
			roomBody = m.Markdown
		} else {
			personBody = m.Markdown
		}
	}
	if roomBody != "New Ticket: Printer down (custom)\n\n---" {
		t.Errorf("room body = %q", roomBody)
	}
	if !strings.HasPrefix(personBody, "**New Ticket:** [42](") || !strings.Contains(personBody, "**Company:** Acme") {
		t.Errorf("person should get the default body: %q", personBody)
	}
}

func TestRedirectRoomTakesEveryMessageEvenInDryRun(t *testing.T) {
	fx := newSendFixture(nil)
	austin := 2
	fx.svc.Cfg.RedirectRoomID = &austin
	outs, err := fx.svc.Send(context.Background(), SendRequest{
		Ticket:  fullTicket(),
		DryRun:  true,
		Intents: []workflow.NotifyIntent{roomIntent("rooms", 1), ownerIntent("people")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.sender.sent) == 0 {
		t.Fatal("redirect must send even under dry run")
	}
	for _, m := range fx.sender.sent {
		if m.RoomID != "wx-Austin" || m.ToPersonEmail != "" {
			t.Errorf("message went to %+v, want the Austin room", m)
		}
		if !strings.Contains(m.Markdown, "Redirected") || !strings.Contains(m.Markdown, "intended for") {
			t.Errorf("missing redirect prefix: %q", m.Markdown)
		}
	}
	by := byRecipient(outs)
	if o := by[1]; o.Result != ResultSent || o.RedirectedTo != "Austin" || o.Recipient.Name != "Dallas" {
		t.Errorf("room outcome = %+v", o)
	}
	// The stored notification still names the intended recipient.
	for _, n := range fx.notifs.inserted {
		if n.RecipientID != nil && *n.RecipientID == austin {
			t.Error("notification record must keep the intended recipient, not the redirect room")
		}
	}
}

func TestNobodyToNotifyIsRecorded(t *testing.T) {
	fx := newSendFixture(nil)
	ft := fullTicket()
	// bob is the only person on the ticket and wrote the note
	ft.Owner = &models.Member{ID: 2, PrimaryEmail: "bob@x.com"}
	ft.Resources = []*models.Member{{ID: 2, PrimaryEmail: "bob@x.com"}}
	outs, err := fx.svc.Send(context.Background(), SendRequest{Ticket: ft, Intents: []workflow.NotifyIntent{ownerIntent("people")}})
	if err != nil {
		t.Fatal(err)
	}
	if len(outs) != 1 || outs[0].Result != ResultNoRecipients || outs[0].Step.Title != "people" || !strings.Contains(outs[0].Reason, "wrote the triggering note") {
		t.Fatalf("outcomes = %+v", outs)
	}
	if len(fx.sender.sent) != 0 {
		t.Fatal("nothing should be sent")
	}
}

func TestDefaultMessageCarriesTheChangedLine(t *testing.T) {
	fx := newSendFixture(nil)
	changes := []models.FieldChange{{Field: "status", Old: "New", New: "Assigned"}}
	_, err := fx.svc.Send(context.Background(), SendRequest{Ticket: fullTicket(), Intents: []workflow.NotifyIntent{roomIntent("rooms", 1)}, Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	if len(fx.sender.sent) != 1 || !strings.Contains(fx.sender.sent[0].Markdown, "**Changed:** Status: New → Assigned") {
		t.Fatalf("sent = %+v", fx.sender.sent)
	}

	// a new ticket has nothing to compare against, even if a caller passes changes
	fx = newSendFixture(nil)
	_, _ = fx.svc.Send(context.Background(), SendRequest{Ticket: fullTicket(), IsNew: true, Intents: []workflow.NotifyIntent{roomIntent("rooms", 1)}, Changes: changes})
	if strings.Contains(fx.sender.sent[0].Markdown, "Changed:") {
		t.Fatal("new-ticket message must not carry a Changed line")
	}

	// custom messages opt in with the placeholder
	fx = newSendFixture(nil)
	in := roomIntent("rooms", 1)
	in.Action.Message = "{{ticket.id}}: {{changes}}"
	_, _ = fx.svc.Send(context.Background(), SendRequest{Ticket: fullTicket(), Intents: []workflow.NotifyIntent{in}, Changes: changes})
	if !strings.HasPrefix(fx.sender.sent[0].Markdown, "42: Status: New → Assigned") {
		t.Fatalf("custom = %q", fx.sender.sent[0].Markdown)
	}
}
