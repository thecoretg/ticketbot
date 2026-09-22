package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/msgtemplate"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/simulate"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowHandler struct {
	Service  *workflow.Service
	CW       *cwsvc.Service
	Notifier *notifier.Service
	Sim      *simulate.Service
}

func NewWorkflowHandler(svc *workflow.Service, cw *cwsvc.Service, ns *notifier.Service, sim *simulate.Service) *WorkflowHandler {
	return &WorkflowHandler{Service: svc, CW: cw, Notifier: ns, Sim: sim}
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

	out, err := h.Sim.Simulate(r.Context(), simulate.Request{WorkflowID: id, TicketID: req.TicketID, AsNew: req.AsNew, Draft: req.Workflow})
	if err != nil {
		var bm *simulate.BoardMismatchError
		switch {
		case errors.Is(err, models.ErrTicketNotFound):
			notFoundError(w, err)
		case errors.As(err, &bm):
			badPayloadError(w, err)
		default:
			h.workflowError(w, err)
		}
		return
	}
	outputJSON(w, out)
}

type previewMessageRequest struct {
	Message  string `json:"message"`
	TicketID int    `json:"ticket_id"`
	AsNew    bool   `json:"as_new"`
}

// PreviewMessage handles POST /workflows/preview-message: it renders a notify message template
// against a stored ticket so an editor can see what the placeholders produce before saving.
func (h *WorkflowHandler) PreviewMessage(w http.ResponseWriter, r *http.Request) {
	var req previewMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		badPayloadError(w, err)
		return
	}
	if req.TicketID == 0 {
		badPayloadError(w, errors.New("ticket_id is required"))
		return
	}
	if err := msgtemplate.Validate(req.Message); err != nil {
		writeJSON(w, http.StatusBadRequest, M{"error": err.Error()})
		return
	}
	detail, err := h.CW.GetTicketDetail(r.Context(), req.TicketID)
	if err != nil {
		if errors.Is(err, models.ErrTicketNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}
	rendered := msgtemplate.Render(req.Message, msgtemplate.Context{
		Ticket:     simulate.FullTicketFromDetail(detail),
		StepTitle:  "Preview",
		IsNew:      req.AsNew,
		CompanyID:  h.Notifier.CWCompanyID,
		MaxNoteLen: h.Notifier.Cfg.MaxMessageLength,
	})
	outputJSON(w, M{"rendered": rendered, "ticket_id": req.TicketID})
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

	ev, err := h.Sim.Evaluate(r.Context(), req.Condition, req.TicketID)
	if err != nil {
		var se *cwquery.SyntaxError
		switch {
		case errors.As(err, &se):
			writeJSON(w, http.StatusBadRequest, M{"error": se.Error(), "pos": se.Pos})
		case errors.Is(err, models.ErrTicketNotFound):
			notFoundError(w, err)
		default:
			internalServerError(w, err)
		}
		return
	}
	outputJSON(w, ev)
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
