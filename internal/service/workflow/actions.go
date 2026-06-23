package workflow

import (
	"context"
	"fmt"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/models"
)

// AddNote posts a note to the ticket. It is NOT idempotent (re-running posts a
// duplicate), so the engine guards it with run-once markers and the author gate.
type AddNote struct{}

type AddNoteParams struct {
	Text              string `json:"text" tmpl:"text"`
	Internal          bool   `json:"internal"`
	DetailDescription bool   `json:"detail_description"`
	Resolution        bool   `json:"resolution"`
}

func (AddNoteParams) isParams() {}

func (AddNote) ActionType() string { return models.WorkflowActionAddNote }
func (AddNote) NewParams() Params  { return &AddNoteParams{} }
func (AddNote) Idempotent() bool   { return false }

func (AddNote) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*AddNoteParams)
	if pp.Text == "" {
		return Change{Field: "note"}, nil // nothing to post
	}

	// The member is intentionally left unset: ConnectWise attributes the note to
	// the API key's member, which is the same member the loop guard keys off of
	// (Cfg.CwBotMemberIdentifier). Setting member/identifier explicitly is rejected
	// by CW unless it exactly matches an existing member, so we let it default.
	note := &psa.ServiceTicketNote{
		TicketID:              t.ID,
		Text:                  pp.Text,
		DetailDescriptionFlag: pp.DetailDescription,
		InternalFlag:          pp.Internal,
		InternalAnalysisFlag:  pp.Internal,
		ResolutionFlag:        pp.Resolution,
	}

	if _, err := x.CW.PostServiceTicketNote(ctx, note, t.ID); err != nil {
		return Change{}, fmt.Errorf("posting note: %w", err)
	}

	return Change{Applied: true, Field: "note", To: pp.Text}, nil
}
