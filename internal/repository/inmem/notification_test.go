package inmem

import (
	"context"
	"testing"

	"github.com/thecoretg/ticketbot/internal/models"
)

func TestNotificationRepo_ExistsForNote(t *testing.T) {
	ctx := context.Background()
	r := NewNotificationRepo(nil)
	noteID := 10101010
	n := models.TicketNotification{
		TicketID:     12345,
		TicketNoteID: &noteID,
		Sent:         true,
	}

	noti, _ := r.Insert(ctx, n)
	t.Logf("inserted note; id: %d, note_id: %d", noti.ID, *noti.TicketNoteID)
	exists, _ := r.ExistsForNote(ctx, noteID)
	if !exists {
		t.Errorf("note not found with id %d", noteID)
	}
}
