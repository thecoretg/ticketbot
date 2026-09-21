package handlers

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/workflow"
	"github.com/thecoretg/ticketbot/models"
)

// RunsHandler serves the workflow results page: run summaries and one run's detail.
type RunsHandler struct {
	Runs      repos.WorkflowRunRepository
	Events    repos.TicketEventRepository
	Boards    repos.BoardRepository
	Workflows *workflow.Service
}

func NewRunsHandler(runs repos.WorkflowRunRepository, events repos.TicketEventRepository, boards repos.BoardRepository, wf *workflow.Service) *RunsHandler {
	return &RunsHandler{Runs: runs, Events: events, Boards: boards, Workflows: wf}
}

// List handles GET /workflows/runs?board_id=&outcome=&ticket_id=&from=&to=&before=&limit=.
// Dates are RFC 3339; from is inclusive, to and before exclusive.
func (h *RunsHandler) List(w http.ResponseWriter, r *http.Request) {
	var f models.RunFilter
	var err error
	if f.BoardID, err = optionalIntQuery(r, "board_id"); err != nil {
		badQueryError(w, err)
		return
	}
	if f.TicketID, err = optionalIntQuery(r, "ticket_id"); err != nil {
		badQueryError(w, err)
		return
	}
	if f.Limit, err = intQueryDefault(r, "limit", 100); err != nil {
		badQueryError(w, err)
		return
	}
	f.Outcome = models.RunOutcome(r.URL.Query().Get("outcome"))
	for _, q := range []struct {
		name string
		dst  **time.Time
	}{{"from", &f.From}, {"to", &f.To}, {"before", &f.Before}} {
		if s := r.URL.Query().Get(q.name); s != "" {
			t, err := time.Parse(time.RFC3339, s)
			if err != nil {
				badQueryError(w, fmt.Errorf("%s must be RFC 3339: %w", q.name, err))
				return
			}
			*q.dst = &t
		}
	}

	runs, err := h.Runs.List(r.Context(), f)
	if err != nil {
		internalServerError(w, err)
		return
	}
	h.nameBoards(r, runs)
	outputJSON(w, runs)
}

// Get handles GET /workflows/runs/{run_id}: the summary, its events and the workflow as it is now.
func (h *RunsHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("run_id")
	run, err := h.Runs.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, models.ErrRunNotFound) {
			notFoundError(w, err)
			return
		}
		internalServerError(w, err)
		return
	}
	h.nameBoards(r, []*models.WorkflowRun{run})

	events, err := h.Events.ListByRun(r.Context(), id)
	if err != nil {
		internalServerError(w, err)
		return
	}
	d := models.RunDetail{Run: *run, Events: events}
	if run.WorkflowID != nil {
		if wf, err := h.Workflows.Get(r.Context(), *run.WorkflowID); err == nil {
			d.Workflow = wf
		}
	}
	outputJSON(w, d)
}

// nameBoards fills BoardName from the synced board table; a missing board stays nameless.
func (h *RunsHandler) nameBoards(r *http.Request, runs []*models.WorkflowRun) {
	names := map[int]string{}
	for _, run := range runs {
		if _, seen := names[run.BoardID]; !seen {
			names[run.BoardID] = ""
			if b, err := h.Boards.Get(r.Context(), run.BoardID); err == nil {
				names[run.BoardID] = b.Name
			}
		}
		run.BoardName = names[run.BoardID]
	}
}
