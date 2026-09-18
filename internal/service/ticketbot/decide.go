package ticketbot

import (
	"encoding/json"
	"strings"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/ticketdiff"
	"github.com/thecoretg/ticketbot/models"
)

// decision is what an intake pass concluded by comparing the stored ticket with ConnectWise.
type decision struct {
	IsNew   bool
	Changes []models.FieldChange
	NewNote bool
}

// Changed reports whether anything worth recording happened.
func (d decision) Changed() bool {
	return d.IsNew || len(d.Changes) > 0 || d.NewNote
}

// decide compares a stored ticket (nil when never seen) against a fresh fetch. Rows stored before
// raw JSON existed are compared on legacy fields only.
func decide(stored *models.Ticket, f *cwsvc.Fetched) decision {
	if stored == nil {
		return decision{IsNew: true, NewNote: f.Note != nil && f.Note.ID != 0}
	}

	var (
		old    *psa.Ticket
		fields = ticketdiff.CuratedFields
	)
	if len(stored.Raw) > 0 {
		old = &psa.Ticket{}
		if err := json.Unmarshal(stored.Raw, old); err != nil {
			old = nil
		}
	}
	if old == nil {
		old = ticketdiff.FromStored(stored)
		fields = ticketdiff.LegacyFields
	}

	d := decision{Changes: ticketdiff.Diff(old, f.Ticket, fields)}
	if f.Note != nil && f.Note.ID != 0 {
		d.NewNote = stored.LatestNoteID == nil || *stored.LatestNoteID != f.Note.ID
	}

	return d
}

// changePayload builds the created/updated event payload.
func changePayload(d decision, f *cwsvc.Fetched, previewLen int) models.ChangePayload {
	p := models.ChangePayload{
		Changes:   d.Changes,
		UpdatedBy: f.Ticket.Info.UpdatedBy,
	}
	if p.Changes == nil {
		p.Changes = []models.FieldChange{}
	}
	if d.NewNote && f.Note != nil {
		p.NewNote = notePayload(f.Note, previewLen)
	}

	return p
}

func notePayload(n *psa.ServiceTicketNote, previewLen int) *models.NotePayload {
	if n == nil {
		return nil
	}

	author, authorName := n.Member.Identifier, n.Member.Name
	if author == "" && n.Contact.ID != 0 {
		authorName = n.Contact.Name
	}

	return &models.NotePayload{
		ID:               n.ID,
		AuthorIdentifier: author,
		AuthorName:       authorName,
		Internal:         n.InternalAnalysisFlag,
		Preview:          preview(n.Text, previewLen),
	}
}

func preview(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= n {
		return s
	}

	return string(r[:n]) + "..."
}
