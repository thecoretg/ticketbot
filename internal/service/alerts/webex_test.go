package alerts

import (
	"context"
	"strings"
	"testing"

	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type fakeRecips struct {
	repos.WebexRecipientRepository
	byID map[int]*models.WebexRecipient
}

func (f *fakeRecips) Get(_ context.Context, id int) (*models.WebexRecipient, error) {
	if r, ok := f.byID[id]; ok {
		return r, nil
	}
	return nil, models.ErrWebexRecipientNotFound
}

type fakeSender struct {
	repos.MessageSender
	sent []webex.Message
}

func (f *fakeSender) PostMessage(_ context.Context, m *webex.Message) (*webex.Message, error) {
	f.sent = append(f.sent, *m)
	return m, nil
}

func TestWebexAlertPostsToOpsRoomOnlyWhenSet(t *testing.T) {
	room := 5
	sender := &fakeSender{}
	w := &Webex{
		Cfg:        &models.Config{},
		Recipients: &fakeRecips{byID: map[int]*models.WebexRecipient{5: {ID: 5, WebexID: "wx-ops", Name: "Ops", Type: models.RecipientTypeRoom}}},
		Sender:     sender,
	}

	w.Alert(context.Background(), "Down", "details")
	if len(sender.sent) != 0 {
		t.Fatal("no ops room set: nothing should be posted")
	}

	w.Cfg.OpsRoomID = &room
	w.Alert(context.Background(), "Down", "details")
	if len(sender.sent) != 1 || sender.sent[0].RoomID != "wx-ops" || !strings.Contains(sender.sent[0].Markdown, "**Down**") {
		t.Fatalf("sent = %+v", sender.sent)
	}
}
