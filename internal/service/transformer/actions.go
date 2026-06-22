package transformer

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

// cwSummaryMaxLen is Connectwise's hard limit on ticket summary length.
const cwSummaryMaxLen = 100

// UpdateSummary rewrites the ticket summary (title). It is idempotent: if the
// ticket summary already equals the rendered value, it makes no API call.
type UpdateSummary struct{}

type UpdateSummaryParams struct {
	Summary string `json:"summary" tmpl:"summary"`
}

func (UpdateSummaryParams) isParams() {}

func (UpdateSummary) ActionType() string { return "update_summary" }
func (UpdateSummary) NewParams() Params  { return &UpdateSummaryParams{} }
func (UpdateSummary) Idempotent() bool   { return true }

func (UpdateSummary) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*UpdateSummaryParams)

	summary := pp.Summary
	if len(summary) > cwSummaryMaxLen {
		slog.Warn("transformer: truncating summary to Connectwise limit", "ticket_id", t.ID, "limit", cwSummaryMaxLen)
		summary = summary[:cwSummaryMaxLen]
	}

	if summary == "" || t.Summary == summary {
		return Change{Field: "summary"}, nil // no-op (idempotent / empty)
	}

	from := t.Summary
	ops := []psa.PatchOp{{Op: "replace", Path: "/summary", Value: summary}}
	if _, err := x.CW.PatchTicket(ctx, t.ID, ops); err != nil {
		return Change{}, fmt.Errorf("patching summary: %w", err)
	}

	t.Summary = summary // keep in-memory ticket current for later rules
	return Change{Applied: true, Field: "summary", From: from, To: summary}, nil
}

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

func (AddNote) ActionType() string { return "add_note" }
func (AddNote) NewParams() Params  { return &AddNoteParams{} }
func (AddNote) Idempotent() bool   { return false }

func (AddNote) Apply(ctx context.Context, x *Exec, t *psa.Ticket, p Params) (Change, error) {
	pp := p.(*AddNoteParams)
	if pp.Text == "" {
		return Change{Field: "note"}, nil // nothing to post
	}

	note := &psa.ServiceTicketNote{
		TicketID:              t.ID,
		Text:                  pp.Text,
		DetailDescriptionFlag: pp.DetailDescription,
		InternalFlag:          pp.Internal,
		InternalAnalysisFlag:  pp.Internal,
		ResolutionFlag:        pp.Resolution,
	}
	// Author the note as the bot so its resulting webhook is detectable (loop prevention).
	note.Member.Identifier = x.BotMemberIdentifier

	if _, err := x.CW.PostServiceTicketNote(ctx, note, t.ID); err != nil {
		return Change{}, fmt.Errorf("posting note: %w", err)
	}

	return Change{Applied: true, Field: "note", To: pp.Text}, nil
}
