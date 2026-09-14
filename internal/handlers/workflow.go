package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

type WorkflowHandler struct {
	Service *workflow.Service
	CW      *cwsvc.Service
}

func NewWorkflowHandler(svc *workflow.Service, cw *cwsvc.Service) *WorkflowHandler {
	return &WorkflowHandler{Service: svc, CW: cw}
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

	doc := cwquery.NewDocument(t, note, nil)
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
