package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/msgtemplate"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowHandler struct {
	Service  *workflow.Service
	CW       *cwsvc.Service
	Notifier *notifier.Service
	Lists    workflow.ListLoader // admin lists for `in list` conditions; may be nil
}

func NewWorkflowHandler(svc *workflow.Service, cw *cwsvc.Service, ns *notifier.Service, lists workflow.ListLoader) *WorkflowHandler {
	return &WorkflowHandler{Service: svc, CW: cw, Notifier: ns, Lists: lists}
}

// Fields handles GET /workflows/fields.
func (h *WorkflowHandler) Fields(w http.ResponseWriter, r *http.Request) {
	outputJSON(w, workflow.ConditionFields)
}

// Placeholders handles GET /workflows/placeholders: the tokens a notify message may use.
func (h *WorkflowHandler) Placeholders(w http.ResponseWriter, r *http.Request) {
	outputJSON(w, msgtemplate.Placeholders)
}

type simulateRequest struct {
	TicketID int  `json:"ticket_id"`
	AsNew    bool `json:"as_new"`
	// Workflow, when present, is an unsaved draft to simulate instead of the stored workflow.
	Workflow *models.Workflow `json:"workflow,omitempty"`
}

type simulateResponse struct {
	Source     string                      `json:"source"` // stored | live
	Workflow   models.WorkflowPayload      `json:"workflow"`
	Actions    []models.ActionPayload      `json:"actions"`
	Recipients []notifier.RecipientPreview `json:"recipients"`
}

// Simulate handles POST /workflows/:id/simulate. It runs the workflow (or a posted draft) against a
// ticket snapshot as a dry run, resolves notification recipients, and returns what would happen.
// The workflow payload's steps are the path the canvas highlights, in order. Nothing is written to
// ConnectWise, Webex, or the ticket history.
func (h *WorkflowHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req simulateRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.TicketID == 0 {
		badPayloadError(w, errors.New("ticket_id is required"))
		return
	}

	ctx := r.Context()
	wf, err := h.Service.Get(ctx, id)
	if err != nil {
		h.workflowError(w, err)
		return
	}
	if req.Workflow != nil {
		draft := req.Workflow
		draft.ID, draft.BoardID, draft.BoardName = wf.ID, wf.BoardID, wf.BoardName
		if errs := h.Service.Validate(ctx, draft); len(errs) > 0 {
			h.workflowError(w, errs)
			return
		}
		wf = draft
	}

	t, note, live, err := h.CW.TicketSnapshot(ctx, req.TicketID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}
	if t.Board.ID != wf.BoardID {
		badPayloadError(w, fmt.Errorf("ticket %d is on board %d, not this workflow's board %d", req.TicketID, t.Board.ID, wf.BoardID))
		return
	}

	engine := workflow.NewEngine(noopCW{})
	engine.Lists = h.Lists
	res, err := engine.Run(ctx, wf, workflow.Input{
		Ticket:      t,
		TriggerNote: note,
		IsNew:       req.AsNew,
		NewNote:     note != nil,
		DryRun:      true,
	})
	if err != nil {
		internalServerError(w, err)
		return
	}

	out := simulateResponse{
		Source:     map[bool]string{true: "live", false: "stored"}[live],
		Workflow:   workflowRunPayload(wf, res),
		Actions:    []models.ActionPayload{},
		Recipients: []notifier.RecipientPreview{},
	}
	for _, a := range res.Actions {
		out.Actions = append(out.Actions, a.Payload())
	}

	if len(res.Notifies) > 0 {
		detail, err := h.CW.GetTicketDetail(ctx, req.TicketID)
		if err != nil {
			internalServerError(w, err)
			return
		}
		ft := &models.FullTicket{
			Ticket:     detail.Ticket.Ticket,
			Board:      detail.Board,
			Status:     detail.Status,
			Company:    detail.Company,
			Contact:    detail.Contact,
			Owner:      detail.Owner,
			LatestNote: detail.LatestNote,
			Resources:  detail.Resources,
		}
		recips, err := h.Notifier.PreviewRecipients(ctx, ft, req.AsNew, res.Notifies)
		if err != nil {
			internalServerError(w, err)
			return
		}
		if recips != nil {
			out.Recipients = recips
		}
	}

	outputJSON(w, out)
}

func workflowRunPayload(wf *models.Workflow, res *workflow.Result) models.WorkflowPayload {
	p := models.WorkflowPayload{WorkflowID: &wf.ID, WorkflowName: wf.Name, Found: true, Enabled: wf.Enabled, Event: res.Event, Steps: []models.StepPayload{}}
	for _, st := range res.Steps {
		p.Steps = append(p.Steps, st.Payload())
	}
	return p
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

func (h *WorkflowHandler) List(w http.ResponseWriter, r *http.Request) {
	ws, err := h.Service.List(r.Context())
	if err != nil {
		internalServerError(w, err)
		return
	}
	if ws == nil {
		ws = []*models.Workflow{}
	}

	outputJSON(w, ws)
}

func (h *WorkflowHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	wf, err := h.Service.Get(r.Context(), id)
	if err != nil {
		h.workflowError(w, err)
		return
	}

	outputJSON(w, wf)
}

func (h *WorkflowHandler) GetByBoard(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	wf, err := h.Service.GetByBoard(r.Context(), id)
	if err != nil {
		h.workflowError(w, err)
		return
	}

	outputJSON(w, wf)
}

// createWorkflowRequest mirrors models.Workflow but lets an omitted "enabled" default to true.
type createWorkflowRequest struct {
	BoardID int           `json:"board_id"`
	Name    string        `json:"name"`
	Enabled *bool         `json:"enabled"`
	DryRun  bool          `json:"dry_run"`
	Nodes   []models.Node `json:"nodes"`
	Edges   []models.Edge `json:"edges"`
}

func (h *WorkflowHandler) Create(w http.ResponseWriter, r *http.Request) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	var req createWorkflowRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.BoardID == 0 {
		badPayloadError(w, errors.New("board_id is required"))
		return
	}

	wf := &models.Workflow{
		BoardID: req.BoardID,
		Name:    req.Name,
		Enabled: req.Enabled == nil || *req.Enabled,
		DryRun:  req.DryRun,
		Nodes:   req.Nodes,
		Edges:   req.Edges,
	}

	created, err := h.Service.Create(r.Context(), wf)
	if err != nil {
		h.workflowError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

func (h *WorkflowHandler) Replace(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	wf, ok := bindWorkflow(w, r)
	if !ok {
		return
	}

	updated, err := h.Service.Replace(r.Context(), id, wf)
	if err != nil {
		h.workflowError(w, err)
		return
	}

	outputJSON(w, updated)
}

func (h *WorkflowHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := convertID(r)
	if err != nil {
		badIntError(w, r)
		return
	}

	if err := h.Service.Delete(r.Context(), id); err != nil {
		h.workflowError(w, err)
		return
	}

	resultJSON(w, "workflow deleted")
}

type conditionRequest struct {
	Condition string `json:"condition"`
	TicketID  int    `json:"ticket_id"`
}

type validateConditionResponse struct {
	Valid bool   `json:"valid"`
	Error string `json:"error,omitempty"`
	Pos   *int   `json:"pos,omitempty"`
}

// ValidateCondition handles POST /workflows/validate-condition.
func (h *WorkflowHandler) ValidateCondition(w http.ResponseWriter, r *http.Request) {
	var req conditionRequest
	if err := decodeJSON(r, &req); err != nil {
		badPayloadError(w, err)
		return
	}

	expr, err := cwquery.Parse(req.Condition)
	if err != nil {
		var se *cwquery.SyntaxError
		if errors.As(err, &se) {
			pos := se.Pos
			outputJSON(w, validateConditionResponse{Valid: false, Error: se.Msg, Pos: &pos})
			return
		}
		outputJSON(w, validateConditionResponse{Valid: false, Error: err.Error()})
		return
	}

	problems, err := h.Service.ValidateListRefs(r.Context(), expr)
	if err != nil {
		internalServerError(w, err)
		return
	}
	if len(problems) > 0 {
		pos := problems[0].Pos
		outputJSON(w, validateConditionResponse{Valid: false, Error: problems[0].Msg, Pos: &pos})
		return
	}

	outputJSON(w, validateConditionResponse{Valid: true})
}

type parseConditionResponse struct {
	Valid bool          `json:"valid"`
	Error string        `json:"error,omitempty"`
	Pos   *int          `json:"pos,omitempty"`
	Expr  *cwquery.Node `json:"expr,omitempty"`
}

// ParseCondition handles POST /workflows/parse-condition: returns the condition's syntax tree so
// the dashboard can rebuild visual builder rows from hand-written text.
func (h *WorkflowHandler) ParseCondition(w http.ResponseWriter, r *http.Request) {
	var req conditionRequest
	if err := decodeJSON(r, &req); err != nil {
		badPayloadError(w, err)
		return
	}

	expr, err := cwquery.Parse(req.Condition)
	if err != nil {
		var se *cwquery.SyntaxError
		if errors.As(err, &se) {
			pos := se.Pos
			outputJSON(w, parseConditionResponse{Valid: false, Error: se.Msg, Pos: &pos})
			return
		}
		outputJSON(w, parseConditionResponse{Valid: false, Error: err.Error()})
		return
	}

	outputJSON(w, parseConditionResponse{Valid: true, Expr: cwquery.ToNode(expr)})
}

type evaluateConditionResponse struct {
	Matches  bool           `json:"matches"`
	Source   string         `json:"source"` // stored | live
	Document map[string]any `json:"document"`
}

// EvaluateCondition handles POST /workflows/evaluate-condition: runs a condition against a stored
// ticket so the dashboard can test rules before saving them.
func (h *WorkflowHandler) EvaluateCondition(w http.ResponseWriter, r *http.Request) {
	var req conditionRequest
	if err := decodeJSON(r, &req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.TicketID == 0 {
		badPayloadError(w, errors.New("ticket_id is required"))
		return
	}

	q, err := cwquery.Compile(req.Condition)
	if err != nil {
		var se *cwquery.SyntaxError
		if errors.As(err, &se) {
			writeJSON(w, http.StatusBadRequest, M{"error": se.Error(), "pos": se.Pos})
			return
		}
		badPayloadError(w, err)
		return
	}

	t, note, live, err := h.CW.TicketSnapshot(r.Context(), req.TicketID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}

	env, err := workflow.LoadEnv(r.Context(), h.Lists, q.Expr)
	if err != nil {
		internalServerError(w, err)
		return
	}
	doc := cwquery.NewDocument(t, note, cwquery.Changes{NewNote: note != nil})
	matches, err := q.EvalEnv(doc, env)
	if err != nil {
		internalServerError(w, err)
		return
	}

	source := "stored"
	if live {
		source = "live"
	}
	outputJSON(w, evaluateConditionResponse{Matches: matches, Source: source, Document: doc})
}

// bindWorkflow decodes a workflow body, rejecting unknown fields so the editor cannot store junk.
func bindWorkflow(w http.ResponseWriter, r *http.Request) (*models.Workflow, bool) {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	wf := &models.Workflow{}
	if err := dec.Decode(wf); err != nil {
		badPayloadError(w, err)
		return nil, false
	}

	return wf, true
}

func (h *WorkflowHandler) workflowError(w http.ResponseWriter, err error) {
	var verrs models.ValidationErrors
	switch {
	case errors.Is(err, models.ErrWorkflowNotFound), errors.Is(err, models.ErrBoardNotFound):
		notFoundError(w, err)
	case errors.Is(err, models.ErrWorkflowExistsForBoard):
		conflictError(w, err)
	case errors.As(err, &verrs):
		writeJSON(w, http.StatusBadRequest, M{"error": "validation failed", "details": verrs})
	default:
		internalServerError(w, fmt.Errorf("workflow: %w", err))
	}
}
