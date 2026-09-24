// Package simulate runs a workflow, or a single condition, against a stored ticket without
// writing anything. The dashboard's simulate and evaluate-condition endpoints and the MCP tools
// share it, so both report the same thing.
package simulate

import (
	"context"
	"errors"
	"fmt"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

// BoardMismatchError is returned when the ticket is not on the workflow's board.
type BoardMismatchError struct {
	TicketID, TicketBoard, WorkflowBoard int
}

func (e *BoardMismatchError) Error() string {
	return fmt.Sprintf("ticket %d is on board %d, not this workflow's board %d", e.TicketID, e.TicketBoard, e.WorkflowBoard)
}

type Service struct {
	Workflows *workflow.Service
	CW        *cwsvc.Service
	Notifier  *notifier.Service
	Lists     workflow.ListLoader // admin lists for `in list` conditions; may be nil
}

// Request names the stored workflow to run and the ticket to run it against. Draft, when set, is
// an unsaved workflow simulated in place of the stored one; it inherits the stored identity.
type Request struct {
	WorkflowID int
	TicketID   int
	AsNew      bool
	Draft      *models.Workflow
}

// Result is what would have happened: the path taken, every action's outcome and who each
// notify would have reached.
type Result struct {
	Source     string                      `json:"source"` // stored | live
	Workflow   models.WorkflowPayload      `json:"workflow"`
	Actions    []models.ActionPayload      `json:"actions"`
	Recipients []notifier.RecipientPreview `json:"recipients"`
}

// Simulate runs the workflow as a dry run against a ticket snapshot. Nothing is written to
// ConnectWise, Webex, or the ticket history. Errors: models.ErrWorkflowNotFound,
// models.ErrTicketNotFound, models.ValidationErrors (bad draft), *BoardMismatchError.
func (s *Service) Simulate(ctx context.Context, req Request) (*Result, error) {
	wf, err := s.Workflows.Get(ctx, req.WorkflowID)
	if err != nil {
		return nil, err
	}
	if req.Draft != nil {
		draft := req.Draft
		draft.ID, draft.BoardID, draft.BoardName = wf.ID, wf.BoardID, wf.BoardName
		if errs := s.Workflows.Validate(ctx, draft); len(errs) > 0 {
			return nil, errs
		}
		wf = draft
	}

	t, note, live, err := s.CW.TicketSnapshot(ctx, req.TicketID)
	if err != nil {
		return nil, err
	}
	if t.Board.ID != wf.BoardID {
		return nil, &BoardMismatchError{TicketID: req.TicketID, TicketBoard: t.Board.ID, WorkflowBoard: wf.BoardID}
	}

	engine := workflow.NewEngine(noopCW{})
	engine.Lists = s.Lists
	res, err := engine.Run(ctx, wf, workflow.Input{
		Ticket:      t,
		TriggerNote: note,
		IsNew:       req.AsNew,
		NewNote:     note != nil,
		DryRun:      true,
	})
	if err != nil {
		return nil, fmt.Errorf("running workflow: %w", err)
	}

	out := &Result{
		Source:     source(live),
		Workflow:   RunPayload(wf, res),
		Actions:    []models.ActionPayload{},
		Recipients: []notifier.RecipientPreview{},
	}
	for _, a := range res.Actions {
		out.Actions = append(out.Actions, a.Payload())
	}
	if len(res.Notifies) > 0 {
		detail, err := s.CW.GetTicketDetail(ctx, req.TicketID)
		if err != nil {
			return nil, fmt.Errorf("loading ticket detail: %w", err)
		}
		recips, err := s.Notifier.PreviewRecipients(ctx, FullTicketFromDetail(detail), req.AsNew, res.Notifies)
		if err != nil {
			return nil, fmt.Errorf("resolving recipients: %w", err)
		}
		if recips != nil {
			out.Recipients = recips
		}
	}
	return out, nil
}

// Evaluation is one condition's verdict against a ticket, with the document it was judged on.
type Evaluation struct {
	Matches  bool           `json:"matches"`
	Source   string         `json:"source"` // stored | live
	Document map[string]any `json:"document"`
}

// Evaluate compiles a condition and runs it against a ticket snapshot. Errors:
// *cwquery.SyntaxError, models.ErrTicketNotFound.
func (s *Service) Evaluate(ctx context.Context, condition string, ticketID int) (*Evaluation, error) {
	if ticketID == 0 {
		return nil, errors.New("ticket_id is required")
	}
	q, err := cwquery.Compile(condition)
	if err != nil {
		return nil, err
	}
	t, note, live, err := s.CW.TicketSnapshot(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	env, err := workflow.LoadEnv(ctx, s.Lists, q.Expr)
	if err != nil {
		return nil, fmt.Errorf("loading lists: %w", err)
	}
	doc := cwquery.NewDocument(t, note, cwquery.Changes{NewNote: note != nil})
	matches, err := q.EvalEnv(doc, env)
	if err != nil {
		return nil, fmt.Errorf("evaluating condition: %w", err)
	}
	return &Evaluation{Matches: matches, Source: source(live), Document: doc}, nil
}

func source(live bool) string {
	if live {
		return "live"
	}
	return "stored"
}

// RunPayload is the workflow half of a run report: the path the canvas highlights, in order.
func RunPayload(wf *models.Workflow, res *workflow.Result) models.WorkflowPayload {
	p := models.WorkflowPayload{WorkflowID: &wf.ID, WorkflowName: wf.Name, Found: true, Enabled: wf.Enabled, Event: res.Event, Steps: []models.StepPayload{}}
	for _, st := range res.Steps {
		p.Steps = append(p.Steps, st.Payload())
	}
	return p
}

// FullTicketFromDetail reshapes a stored ticket detail into what the notifier and message
// templates read.
func FullTicketFromDetail(detail *models.TicketDetail) *models.FullTicket {
	return &models.FullTicket{
		Ticket:     detail.Ticket.Ticket,
		Board:      detail.Board,
		Status:     detail.Status,
		Company:    detail.Company,
		Contact:    detail.Contact,
		Owner:      detail.Owner,
		LatestNote: detail.LatestNote,
		Resources:  detail.Resources,
	}
}

// noopCW satisfies workflow.CWClient for simulations; the engine never writes in dry run, and the
// read methods are only reached after a write, so none of these should be called.
type noopCW struct{}

func (noopCW) GetTicket(context.Context, int, map[string]string) (*psa.Ticket, error) {
	return nil, errors.New("simulation: connectwise reads are disabled")
}
func (noopCW) GetMostRecentTicketNote(context.Context, int) (*psa.ServiceTicketNote, error) {
	return nil, errors.New("simulation: connectwise reads are disabled")
}
func (noopCW) PatchTicket(context.Context, int, []psa.PatchOp) (*psa.Ticket, error) {
	return nil, errors.New("simulation: connectwise writes are disabled")
}
func (noopCW) PostServiceTicketNote(context.Context, *psa.ServiceTicketNote, int) (*psa.ServiceTicketNote, error) {
	return nil, errors.New("simulation: connectwise writes are disabled")
}
