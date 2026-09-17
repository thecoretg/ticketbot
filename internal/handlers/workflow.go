package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
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
}

func NewWorkflowHandler(svc *workflow.Service, cw *cwsvc.Service, ns *notifier.Service) *WorkflowHandler {
	return &WorkflowHandler{Service: svc, CW: cw, Notifier: ns}
}

// Fields handles GET /workflows/fields.
func (h *WorkflowHandler) Fields(c *gin.Context) {
	outputJSON(c, workflow.ConditionFields)
}

// Placeholders handles GET /workflows/placeholders: the tokens a notify message may use.
func (h *WorkflowHandler) Placeholders(c *gin.Context) {
	outputJSON(c, msgtemplate.Placeholders)
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
// Nothing is written to ConnectWise, Webex, or the ticket history.
func (h *WorkflowHandler) Simulate(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()
	var req simulateRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(c, err)
		return
	}
	if req.TicketID == 0 {
		badPayloadError(c, errors.New("ticket_id is required"))
		return
	}

	ctx := c.Request.Context()
	wf, err := h.Service.Get(ctx, id)
	if err != nil {
		h.workflowError(c, err)
		return
	}
	if req.Workflow != nil {
		draft := req.Workflow
		draft.ID, draft.BoardID, draft.BoardName = wf.ID, wf.BoardID, wf.BoardName
		if errs := h.Service.Validate(ctx, draft); len(errs) > 0 {
			h.workflowError(c, errs)
			return
		}
		wf = draft
	}

	t, note, live, err := h.CW.TicketSnapshot(ctx, req.TicketID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}
	if t.Board.ID != wf.BoardID {
		badPayloadError(c, fmt.Errorf("ticket %d is on board %d, not this workflow's board %d", req.TicketID, t.Board.ID, wf.BoardID))
		return
	}

	engine := workflow.NewEngine(noopCW{})
	res, err := engine.Run(ctx, wf, workflow.Input{
		Ticket:      t,
		TriggerNote: note,
		IsNew:       req.AsNew,
		NewNote:     note != nil,
		DryRun:      true,
	})
	if err != nil {
		internalServerError(c, err)
		return
	}

	out := simulateResponse{
		Source:     map[bool]string{true: "live", false: "stored"}[live],
		Workflow:   workflowRunPayload(wf, res),
		Actions:    []models.ActionPayload{},
		Recipients: []notifier.RecipientPreview{},
	}
	for _, a := range res.Actions {
		p := models.ActionPayload{RuleID: a.Rule.RuleID, RuleName: a.Rule.RuleName, Index: a.Index, Kind: string(a.Kind), Result: a.Result, Reason: a.Reason, Output: a.Output}
		if a.Err != nil {
			p.Error = a.Err.Error()
		}
		out.Actions = append(out.Actions, p)
	}

	if len(res.Notifies) > 0 {
		detail, err := h.CW.GetTicketDetail(ctx, req.TicketID)
		if err != nil {
			internalServerError(c, err)
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
			internalServerError(c, err)
			return
		}
		if recips != nil {
			out.Recipients = recips
		}
	}

	outputJSON(c, out)
}

func workflowRunPayload(wf *models.Workflow, res *workflow.Result) models.WorkflowPayload {
	p := models.WorkflowPayload{WorkflowID: &wf.ID, WorkflowName: wf.Name, Found: true, Enabled: wf.Enabled, Rules: []models.RuleOutcomePayload{}}
	for _, r := range res.Rules {
		rp := models.RuleOutcomePayload{RuleID: r.Rule.RuleID, RuleName: r.Rule.RuleName, Matched: r.Matched, Skipped: r.Skipped, Stopped: r.Stopped}
		if r.Err != nil {
			rp.Error = r.Err.Error()
		}
		p.Rules = append(p.Rules, rp)
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

func (h *WorkflowHandler) List(c *gin.Context) {
	ws, err := h.Service.List(c.Request.Context())
	if err != nil {
		internalServerError(c, err)
		return
	}
	if ws == nil {
		ws = []*models.Workflow{}
	}

	outputJSON(c, ws)
}

func (h *WorkflowHandler) Get(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	w, err := h.Service.Get(c.Request.Context(), id)
	if err != nil {
		h.workflowError(c, err)
		return
	}

	outputJSON(c, w)
}

func (h *WorkflowHandler) GetByBoard(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	w, err := h.Service.GetByBoard(c.Request.Context(), id)
	if err != nil {
		h.workflowError(c, err)
		return
	}

	outputJSON(c, w)
}

// createWorkflowRequest mirrors models.Workflow but lets an omitted "enabled" default to true.
type createWorkflowRequest struct {
	BoardID int           `json:"board_id"`
	Name    string        `json:"name"`
	Enabled *bool         `json:"enabled"`
	DryRun  bool          `json:"dry_run"`
	Rules   []models.Rule `json:"rules"`
}

func (h *WorkflowHandler) Create(c *gin.Context) {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()

	var req createWorkflowRequest
	if err := dec.Decode(&req); err != nil {
		badPayloadError(c, err)
		return
	}
	if req.BoardID == 0 {
		badPayloadError(c, errors.New("board_id is required"))
		return
	}

	w := &models.Workflow{
		BoardID: req.BoardID,
		Name:    req.Name,
		Enabled: req.Enabled == nil || *req.Enabled,
		DryRun:  req.DryRun,
		Rules:   req.Rules,
	}

	created, err := h.Service.Create(c.Request.Context(), w)
	if err != nil {
		h.workflowError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

func (h *WorkflowHandler) Replace(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	w, ok := bindWorkflow(c)
	if !ok {
		return
	}

	updated, err := h.Service.Replace(c.Request.Context(), id, w)
	if err != nil {
		h.workflowError(c, err)
		return
	}

	outputJSON(c, updated)
}

func (h *WorkflowHandler) Delete(c *gin.Context) {
	id, err := convertID(c)
	if err != nil {
		badIntError(c)
		return
	}

	if err := h.Service.Delete(c.Request.Context(), id); err != nil {
		h.workflowError(c, err)
		return
	}

	resultJSON(c, "workflow deleted")
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
func (h *WorkflowHandler) ValidateCondition(c *gin.Context) {
	var req conditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badPayloadError(c, err)
		return
	}

	if se := workflow.ValidateCondition(req.Condition); se != nil {
		pos := se.Pos
		outputJSON(c, validateConditionResponse{Valid: false, Error: se.Msg, Pos: &pos})
		return
	}

	outputJSON(c, validateConditionResponse{Valid: true})
}

type parseConditionResponse struct {
	Valid bool          `json:"valid"`
	Error string        `json:"error,omitempty"`
	Pos   *int          `json:"pos,omitempty"`
	Expr  *cwquery.Node `json:"expr,omitempty"`
}

// ParseCondition handles POST /workflows/parse-condition: returns the condition's syntax tree so
// the dashboard can rebuild visual builder rows from hand-written text.
func (h *WorkflowHandler) ParseCondition(c *gin.Context) {
	var req conditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badPayloadError(c, err)
		return
	}

	expr, err := cwquery.Parse(req.Condition)
	if err != nil {
		var se *cwquery.SyntaxError
		if errors.As(err, &se) {
			pos := se.Pos
			outputJSON(c, parseConditionResponse{Valid: false, Error: se.Msg, Pos: &pos})
			return
		}
		outputJSON(c, parseConditionResponse{Valid: false, Error: err.Error()})
		return
	}

	outputJSON(c, parseConditionResponse{Valid: true, Expr: cwquery.ToNode(expr)})
}

type evaluateConditionResponse struct {
	Matches  bool           `json:"matches"`
	Source   string         `json:"source"` // stored | live
	Document map[string]any `json:"document"`
}

// EvaluateCondition handles POST /workflows/evaluate-condition: runs a condition against a stored
// ticket so the dashboard can test rules before saving them.
func (h *WorkflowHandler) EvaluateCondition(c *gin.Context) {
	var req conditionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		badPayloadError(c, err)
		return
	}
	if req.TicketID == 0 {
		badPayloadError(c, errors.New("ticket_id is required"))
		return
	}

	q, err := cwquery.Compile(req.Condition)
	if err != nil {
		var se *cwquery.SyntaxError
		if errors.As(err, &se) {
			c.JSON(http.StatusBadRequest, gin.H{"error": se.Error(), "pos": se.Pos})
			return
		}
		badPayloadError(c, err)
		return
	}

	t, note, live, err := h.CW.TicketSnapshot(c.Request.Context(), req.TicketID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(c, err)
			return
		}
		internalServerError(c, err)
		return
	}

	doc := cwquery.NewDocument(t, note, cwquery.Changes{NewNote: note != nil})
	matches, err := q.Eval(doc)
	if err != nil {
		internalServerError(c, err)
		return
	}

	source := "stored"
	if live {
		source = "live"
	}
	outputJSON(c, evaluateConditionResponse{Matches: matches, Source: source, Document: doc})
}

// bindWorkflow decodes a workflow body, rejecting unknown fields so the editor cannot store junk.
func bindWorkflow(c *gin.Context) (*models.Workflow, bool) {
	dec := json.NewDecoder(c.Request.Body)
	dec.DisallowUnknownFields()

	w := &models.Workflow{}
	if err := dec.Decode(w); err != nil {
		badPayloadError(c, err)
		return nil, false
	}

	return w, true
}

func (h *WorkflowHandler) workflowError(c *gin.Context, err error) {
	var verrs models.ValidationErrors
	switch {
	case errors.Is(err, models.ErrWorkflowNotFound), errors.Is(err, models.ErrBoardNotFound):
		notFoundError(c, err)
	case errors.Is(err, models.ErrWorkflowExistsForBoard):
		conflictError(c, err)
	case errors.As(err, &verrs):
		c.JSON(http.StatusBadRequest, gin.H{"error": "validation failed", "details": verrs})
	default:
		internalServerError(c, fmt.Errorf("workflow: %w", err))
	}
}
