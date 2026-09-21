package alerts

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// Webex posts alerts to the configured ops room and logs them when none is set.
type Webex struct {
	Cfg        *models.Config
	Recipients repos.WebexRecipientRepository
	Sender     repos.MessageSender
}

func (w *Webex) Alert(ctx context.Context, subject, body string) {
	Log{}.Alert(ctx, subject, body)
	if w == nil || w.Cfg == nil || w.Cfg.OpsRoomID == nil || w.Recipients == nil || w.Sender == nil {
		return
	}

	r, err := w.Recipients.Get(ctx, *w.Cfg.OpsRoomID)
	if err != nil {
		slog.Error("alerts: ops room lookup failed", "recipient_id", *w.Cfg.OpsRoomID, "error", err.Error())
		return
	}

	text := fmt.Sprintf("⚠️ **%s**\n\n%s", subject, body)
	var msg webex.Message
	if r.Type == models.RecipientTypePerson && r.Email != nil {
		msg = webex.NewMessageToPerson(*r.Email, text)
	} else {
		msg = webex.NewMessageToRoom(r.WebexID, r.Name, text)
	}
	if _, err := w.Sender.PostMessage(ctx, &msg); err != nil {
		slog.Error("alerts: posting to ops room failed", "room", r.Name, "error", err.Error())
	}
}
