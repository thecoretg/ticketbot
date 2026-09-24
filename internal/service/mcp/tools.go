package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thecoretg/ticketbot/internal/cwquery"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/internal/service/simulate"
	"github.com/thecoretg/ticketbot/models"
)

// Deps is what the tools read from. Each interface is the slice of a service or repo the tools
// use, so tests can fake one without the rest.
type Deps struct {
	Workflows WorkflowReader
	Runs      RunReader
	Events    EventReader
	Tickets   TicketReader
	Lists     ListReader
	Forwards  ForwardReader
	Config    ConfigReader
	Simulator Simulator
	CW        CWLookup
	Webex     WebexReader
	Intake    IntakeReader
	Logs      LogReader
}

type WorkflowReader interface {
	List(ctx context.Context) ([]*models.Workflow, error)
	Get(ctx context.Context, id int) (*models.Workflow, error)
}

type RunReader interface {
	Get(ctx context.Context, runID string) (*models.WorkflowRun, error)
	List(ctx context.Context, f models.RunFilter) ([]*models.WorkflowRun, error)
}

type EventReader interface {
	ListByRun(ctx context.Context, runID string) ([]*models.TicketEvent, error)
}

type TicketReader interface {
	GetTicketDetail(ctx context.Context, id int) (*models.TicketDetail, error)
	ListTicketEvents(ctx context.Context, id int, limit int, beforeID *int64) ([]*models.TicketEvent, error)
}

type ListReader interface {
	List(ctx context.Context) ([]*models.List, error)
	Get(ctx context.Context, id int) (*models.ListDetail, error)
}

type ForwardReader interface {
	ListForwardsFull(ctx context.Context) ([]*models.NotifierForwardFull, error)
}

type ConfigReader interface {
	Get(ctx context.Context) (*models.Config, error)
}

type Simulator interface {
	Simulate(ctx context.Context, req simulate.Request) (*simulate.Result, error)
	Evaluate(ctx context.Context, condition string, ticketID int) (*simulate.Evaluation, error)
}

type CWLookup interface {
	ListBoards(ctx context.Context) ([]*models.Board, error)
	ListStatusesByBoard(ctx context.Context, boardID int) ([]*models.TicketStatus, error)
	ListMembers(ctx context.Context) ([]*models.Member, error)
	ListPriorities(ctx context.Context) ([]*models.Priority, error)
	SearchCompanies(ctx context.Context, f models.CompanySearch) ([]*models.Company, error)
	SearchContacts(ctx context.Context, f models.ContactSearch) ([]*models.Contact, error)
}

type WebexReader interface {
	ListRecipients(ctx context.Context) ([]*models.WebexRecipient, error)
}

type IntakeReader interface {
	Stats(ctx context.Context) (*models.IntakeStats, error)
	List(ctx context.Context, status *models.IntakeStatus, limit int) ([]*models.WebhookIntake, error)
}

type LogReader interface {
	Entries() []logging.LogEntry
}

// toolDef binds a tool to the role floor and scope it needs; add registers it on a Server.
type toolDef struct {
	name    string
	minRole models.Role
	scope   string
	add     func(*sdk.Server)
}

// def builds a toolDef for a typed handler. Every tool here is read-only and idempotent.
func def[In any](name, description string, minRole models.Role, h func(context.Context, In) (any, error)) toolDef {
	full := toolPrefix + name
	return toolDef{
		name:    full,
		minRole: minRole,
		scope:   oauth.ScopeRead,
		add: func(s *sdk.Server) {
			t := &sdk.Tool{
				Name:        full,
				Description: description,
				Annotations: &sdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true},
			}
			sdk.AddTool(s, t, func(ctx context.Context, _ *sdk.CallToolRequest, in In) (*sdk.CallToolResult, any, error) {
				out, err := h(ctx, in)
				if err != nil {
					return nil, nil, err
				}
				if text, ok := out.(string); ok {
					return textResult(text), nil, nil
				}
				return jsonResult(out)
			})
		},
	}
}

// jsonResult renders a value as indented JSON text.
func jsonResult(v any) (*sdk.CallToolResult, any, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, nil, fmt.Errorf("encoding result: %w", err)
	}
	return textResult(string(b)), nil, nil
}

type empty struct{}

type workflowInput struct {
	ID  int  `json:"id" jsonschema:"workflow id, from ticketbot_list_workflows"`
	Raw bool `json:"raw,omitempty" jsonschema:"return the stored workflow document as JSON instead of the readable walk"`
}

type runsInput struct {
	BoardID  int    `json:"board_id,omitempty" jsonschema:"only runs for tickets on this ConnectWise board"`
	TicketID int    `json:"ticket_id,omitempty" jsonschema:"only runs for this ticket"`
	Outcome  string `json:"outcome,omitempty" jsonschema:"clean, errors, nobody_notified or no_trigger"`
	From     string `json:"from,omitempty" jsonschema:"RFC 3339; runs started at or after this time"`
	To       string `json:"to,omitempty" jsonschema:"RFC 3339; runs started before this time"`
	Cursor   string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	Limit    int    `json:"limit,omitempty" jsonschema:"page size, default 20, max 100"`
}

type runInput struct {
	RunID string `json:"run_id" jsonschema:"run id from ticketbot_list_runs or a ticket's history"`
}

type historyInput struct {
	TicketID int    `json:"ticket_id" jsonschema:"ConnectWise ticket id"`
	Cursor   string `json:"cursor,omitempty" jsonschema:"next_cursor from the previous page"`
	Limit    int    `json:"limit,omitempty" jsonschema:"events per page, default 20, max 100"`
}

type listsInput struct {
	ID int `json:"id,omitempty" jsonschema:"list id; when set, returns that list with its items and where it is used"`
}

type simulateInput struct {
	WorkflowID int  `json:"workflow_id" jsonschema:"workflow to run"`
	TicketID   int  `json:"ticket_id" jsonschema:"ticket to run it against; must be on the workflow's board"`
	AsNew      bool `json:"as_new,omitempty" jsonschema:"treat the ticket as just created rather than updated"`
}

type evaluateInput struct {
	Condition string `json:"condition" jsonschema:"a condition in ticketbot's condition language, as shown in a workflow walk"`
	TicketID  int    `json:"ticket_id" jsonschema:"ticket to test it against"`
}

type lookupInput struct {
	Kind    string `json:"kind" jsonschema:"boards, statuses, members, priorities, companies or contacts"`
	BoardID int    `json:"board_id,omitempty" jsonschema:"required for statuses"`
	Query   string `json:"query,omitempty" jsonschema:"name search for companies and contacts"`
	IDs     []int  `json:"ids,omitempty" jsonschema:"exact ids to look up"`
	Limit   int    `json:"limit,omitempty" jsonschema:"max rows for companies and contacts, default 20, max 100"`
}

type intakeInput struct {
	FailedLimit int `json:"failed_limit,omitempty" jsonschema:"how many failed rows to include, default 20, max 100"`
}

type logsInput struct {
	Lines int    `json:"lines,omitempty" jsonschema:"how many of the newest lines, default 200"`
	Level string `json:"level,omitempty" jsonschema:"only this level: DEBUG, INFO, WARN or ERROR"`
}

func (s *Service) toolDefs() []toolDef {
	d := s.deps
	return []toolDef{
		def("list_workflows", "List ticketbot's workflows: one per ConnectWise board, with whether it is enabled, "+
			"in dry run, and how many lanes it has. Use ticketbot_get_workflow for what a workflow does.",
			models.RoleViewer, func(ctx context.Context, _ empty) (any, error) {
				ws, err := d.Workflows.List(ctx)
				if err != nil {
					return nil, err
				}
				out := make([]workflowRow, 0, len(ws))
				for _, w := range ws {
					out = append(out, workflowRowOf(w))
				}
				return out, nil
			}),

		def("get_workflow", "Describe a workflow as a readable walk: each lane's trigger, its 'only when' "+
			"condition, the if branches and the actions on each path. Pass raw=true for the stored "+
			"document (nodes and edges JSON).",
			models.RoleViewer, func(ctx context.Context, in workflowInput) (any, error) {
				w, err := d.Workflows.Get(ctx, in.ID)
				if err != nil {
					return nil, err
				}
				if in.Raw {
					return w, nil
				}
				return describeWorkflow(w), nil
			}),

		def("list_runs", "List workflow runs, newest first: which ticket, which event, outcome, counts of "+
			"actions, writes and notifications. Filter by board, ticket, outcome or time; page with cursor.",
			models.RoleViewer, func(ctx context.Context, in runsInput) (any, error) {
				f := models.RunFilter{Outcome: models.RunOutcome(in.Outcome), Limit: clampLimit(in.Limit)}
				if in.BoardID > 0 {
					f.BoardID = &in.BoardID
				}
				if in.TicketID > 0 {
					f.TicketID = &in.TicketID
				}
				var err error
				if f.From, err = parseTime("from", in.From); err != nil {
					return nil, err
				}
				if f.To, err = parseTime("to", in.To); err != nil {
					return nil, err
				}
				if f.Before, err = parseTime("cursor", in.Cursor); err != nil {
					return nil, err
				}
				runs, err := d.Runs.List(ctx, f)
				if err != nil {
					return nil, err
				}
				page := pageOf[*models.WorkflowRun]{Items: runs}
				if len(runs) == f.Limit {
					page.NextCursor = runs[len(runs)-1].StartedAt.Format(time.RFC3339Nano)
				}
				return page, nil
			}),

		def("get_run", "One workflow run in full: the summary, the path taken through the workflow, every "+
			"action's result and the history events it recorded (notifications, writes, errors).",
			models.RoleViewer, func(ctx context.Context, in runInput) (any, error) {
				run, err := d.Runs.Get(ctx, in.RunID)
				if err != nil {
					return nil, err
				}
				events, err := d.Events.ListByRun(ctx, in.RunID)
				if err != nil {
					return nil, err
				}
				return runDetail{Run: run, Events: compactEvents(events)}, nil
			}),

		def("ticket_history", "What ticketbot has seen and done for one ticket: the ticket as ticketbot last "+
			"stored it, then its history events newest first (webhooks, workflow runs, actions, "+
			"notifications, errors). This is ticketbot's record, not live ConnectWise data.",
			models.RoleViewer, func(ctx context.Context, in historyInput) (any, error) {
				detail, err := d.Tickets.GetTicketDetail(ctx, in.TicketID)
				if err != nil {
					return nil, err
				}
				var before *int64
				if in.Cursor != "" {
					var v int64
					if _, err := fmt.Sscan(in.Cursor, &v); err != nil {
						return nil, fmt.Errorf("cursor must be an event id")
					}
					before = &v
				}
				limit := clampLimit(in.Limit)
				events, err := d.Tickets.ListTicketEvents(ctx, in.TicketID, limit, before)
				if err != nil {
					return nil, err
				}
				out := ticketHistory{Ticket: ticketHead(detail), Events: compactEvents(events)}
				if len(events) == limit {
					out.NextCursor = fmt.Sprint(events[len(events)-1].ID)
				}
				return out, nil
			}),

		def("list_lists", "Ticketbot's lists of companies or contacts, used by 'in list' conditions. With an "+
			"id, returns that list's items and the workflow nodes that reference it.",
			models.RoleViewer, func(ctx context.Context, in listsInput) (any, error) {
				if in.ID > 0 {
					return d.Lists.Get(ctx, in.ID)
				}
				return d.Lists.List(ctx)
			}),

		def("list_forwards", "Notification forwards: who receives another person's Webex notifications, "+
			"with the window and the rules (keep a copy, only if sole resource, public notes only).",
			models.RoleViewer, func(ctx context.Context, _ empty) (any, error) {
				return d.Forwards.ListForwardsFull(ctx)
			}),

		def("get_config", "Ticketbot's settings: master dry run, redirect and ops rooms, write cap, retention, "+
			"business hours, sign-in switches and whether MCP is enabled.",
			models.RoleViewer, func(ctx context.Context, _ empty) (any, error) {
				return d.Config.Get(ctx)
			}),

		def("simulate", "Run a workflow against a ticket ticketbot has stored, as a dry run: returns the path "+
			"taken, each action's result and who each notify would reach. Writes nothing anywhere.",
			models.RoleViewer, func(ctx context.Context, in simulateInput) (any, error) {
				if in.WorkflowID == 0 || in.TicketID == 0 {
					return nil, errors.New("workflow_id and ticket_id are required")
				}
				return d.Simulator.Simulate(ctx, simulate.Request{WorkflowID: in.WorkflowID, TicketID: in.TicketID, AsNew: in.AsNew})
			}),

		def("evaluate_condition", "Test a condition against a stored ticket: returns whether it matches and "+
			"the document (field values) it was judged on, which shows what each condition path resolves to.",
			models.RoleViewer, func(ctx context.Context, in evaluateInput) (any, error) {
				if strings.TrimSpace(in.Condition) == "" {
					return nil, errors.New("condition is required")
				}
				ev, err := d.Simulator.Evaluate(ctx, in.Condition, in.TicketID)
				var se *cwquery.SyntaxError
				if errors.As(err, &se) {
					return nil, fmt.Errorf("condition syntax: %s (position %d)", se.Msg, se.Pos)
				}
				return ev, err
			}),

		def("lookup_ids", "Translate ConnectWise ids to names and back from ticketbot's cache: boards, "+
			"a board's statuses, members, priorities, companies and contacts. Use this to read ids in "+
			"workflow conditions and actions. For live ConnectWise records use the ConnectWise PSA "+
			"connector instead; the ids are the same in both.",
			models.RoleViewer, func(ctx context.Context, in lookupInput) (any, error) {
				return lookup(ctx, d.CW, in)
			}),

		def("list_webex_rooms", "Webex rooms and people ticketbot can notify, with the ticketbot recipient ids "+
			"that notify nodes, forwards and settings refer to.",
			models.RoleViewer, func(ctx context.Context, _ empty) (any, error) {
				return d.Webex.ListRecipients(ctx)
			}),

		def("intake_status", "Admin: the webhook intake queue. Counts per status, when the last webhook "+
			"arrived, and the failed rows with their last error.",
			models.RoleAdmin, func(ctx context.Context, in intakeInput) (any, error) {
				st, err := d.Intake.Stats(ctx)
				if err != nil {
					return nil, err
				}
				failed := models.IntakeFailed
				rows, err := d.Intake.List(ctx, &failed, clampLimit(in.FailedLimit))
				if err != nil {
					return nil, err
				}
				out := intakeStatus{Counts: st.Counts, LastReceivedAt: st.LastReceivedAt}
				for _, r := range rows {
					out.Failed = append(out.Failed, intakeRow{ID: r.ID, TicketID: r.TicketID, Action: string(r.Action), Attempts: r.Attempts, LastError: r.LastError, ReceivedAt: r.ReceivedAt})
				}
				return out, nil
			}),

		def("tail_logs", "Admin: the newest lines of the application log from the in-memory buffer, "+
			"optionally one level only.",
			models.RoleAdmin, func(_ context.Context, in logsInput) (any, error) {
				entries := d.Logs.Entries()
				n := in.Lines
				if n <= 0 {
					n = 200
				}
				level := strings.ToUpper(strings.TrimSpace(in.Level))
				var b strings.Builder
				kept := 0
				for i := len(entries) - 1; i >= 0 && kept < n; i-- {
					e := entries[i]
					if level != "" && !strings.EqualFold(e.Level, level) {
						continue
					}
					kept++
					b.WriteString(formatLogEntry(e))
				}
				if kept == 0 {
					return "no log lines", nil
				}
				// Reverse back to chronological order.
				lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
				slices.Reverse(lines)
				return strings.Join(lines, "\n"), nil
			}),
	}
}

type pageOf[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor,omitempty"`
}

type workflowRow struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	BoardID   int       `json:"board_id"`
	BoardName string    `json:"board_name,omitempty"`
	Enabled   bool      `json:"enabled"`
	DryRun    bool      `json:"dry_run"`
	Lanes     int       `json:"lanes"`
	Nodes     int       `json:"nodes"`
	UpdatedOn time.Time `json:"updated_on"`
}

func workflowRowOf(w *models.Workflow) workflowRow {
	r := workflowRow{ID: w.ID, Name: w.Name, BoardID: w.BoardID, BoardName: w.BoardName, Enabled: w.Enabled, DryRun: w.DryRun, Nodes: len(w.Nodes), UpdatedOn: w.UpdatedOn}
	for _, n := range w.Nodes {
		if n.Kind == models.NodeTrigger {
			r.Lanes++
		}
	}
	return r
}

type runDetail struct {
	Run    *models.WorkflowRun `json:"run"`
	Events []compactEvent      `json:"events"`
}

// compactEvent is a history event with its payload decoded, so the result is one JSON document
// rather than JSON inside strings.
type compactEvent struct {
	ID         int64           `json:"id"`
	Kind       string          `json:"kind"`
	Source     string          `json:"source"`
	DryRun     bool            `json:"dry_run,omitempty"`
	RunID      string          `json:"run_id,omitempty"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

func compactEvents(events []*models.TicketEvent) []compactEvent {
	out := make([]compactEvent, 0, len(events))
	for _, e := range events {
		out = append(out, compactEvent{ID: e.ID, Kind: string(e.Kind), Source: string(e.Source), DryRun: e.DryRun, RunID: e.RunID, OccurredAt: e.OccurredAt, Payload: e.Payload})
	}
	return out
}

type ticketHistory struct {
	Ticket     ticketHeader   `json:"ticket"`
	Events     []compactEvent `json:"events"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type ticketHeader struct {
	ID        int       `json:"id"`
	Summary   string    `json:"summary"`
	Board     string    `json:"board"`
	BoardID   int       `json:"board_id"`
	Status    string    `json:"status"`
	Company   string    `json:"company"`
	Owner     string    `json:"owner,omitempty"`
	Resources *string   `json:"resources,omitempty"`
	Priority  *string   `json:"priority,omitempty"`
	Closed    bool      `json:"closed"`
	UpdatedOn time.Time `json:"updated_on"`
	CWURL     string    `json:"cw_url,omitempty"`
}

func ticketHead(d *models.TicketDetail) ticketHeader {
	t := d.Ticket
	return ticketHeader{
		ID: t.ID, Summary: t.Summary, Board: t.BoardName, BoardID: t.BoardID, Status: t.StatusName, Company: t.CompanyName,
		Owner: t.OwnerName, Resources: t.Resources, Priority: t.PriorityName, Closed: t.ClosedFlag, UpdatedOn: t.UpdatedOn, CWURL: t.CWURL,
	}
}

type intakeStatus struct {
	Counts         map[models.IntakeStatus]int64 `json:"counts"`
	LastReceivedAt *time.Time                    `json:"last_received_at,omitempty"`
	Failed         []intakeRow                   `json:"failed"`
}

type intakeRow struct {
	ID         int64     `json:"id"`
	TicketID   int       `json:"ticket_id"`
	Action     string    `json:"action"`
	Attempts   int       `json:"attempts"`
	LastError  *string   `json:"last_error,omitempty"`
	ReceivedAt time.Time `json:"received_at"`
}

func parseTime(name, v string) (*time.Time, error) {
	if v == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, v)
	if err != nil {
		return nil, fmt.Errorf("%s must be RFC 3339: %w", name, err)
	}
	return &t, nil
}

func formatLogEntry(e logging.LogEntry) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s %-5s %s", e.Time.Format(time.RFC3339), e.Level, e.Message)
	for _, k := range slices.Sorted(mapsKeys(e.Attrs)) {
		fmt.Fprintf(&b, " %s=%v", k, e.Attrs[k])
	}
	b.WriteByte('\n')
	return b.String()
}

func mapsKeys(m map[string]any) func(func(string) bool) {
	return func(yield func(string) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// lookup answers ticketbot_lookup_ids from the ConnectWise cache.
func lookup(ctx context.Context, cw CWLookup, in lookupInput) (any, error) {
	limit := clampLimit(in.Limit)
	switch strings.ToLower(strings.TrimSpace(in.Kind)) {
	case "boards", "board":
		bs, err := cw.ListBoards(ctx)
		return filterIDs(bs, in.IDs, func(b *models.Board) int { return b.ID }), err
	case "statuses", "status":
		if in.BoardID == 0 {
			return nil, errors.New("board_id is required for statuses")
		}
		st, err := cw.ListStatusesByBoard(ctx, in.BoardID)
		return filterIDs(st, in.IDs, func(s *models.TicketStatus) int { return s.ID }), err
	case "members", "member":
		ms, err := cw.ListMembers(ctx)
		return filterIDs(ms, in.IDs, func(m *models.Member) int { return m.ID }), err
	case "priorities", "priority":
		ps, err := cw.ListPriorities(ctx)
		return filterIDs(ps, in.IDs, func(p *models.Priority) int { return p.ID }), err
	case "companies", "company":
		return cw.SearchCompanies(ctx, models.CompanySearch{Query: in.Query, IDs: in.IDs, Limit: limit})
	case "contacts", "contact":
		f := models.ContactSearch{Query: in.Query, IDs: in.IDs, Limit: limit}
		return cw.SearchContacts(ctx, f)
	}
	return nil, fmt.Errorf("kind must be boards, statuses, members, priorities, companies or contacts")
}

func filterIDs[T any](rows []T, ids []int, id func(T) int) []T {
	if len(ids) == 0 {
		return rows
	}
	out := make([]T, 0, len(ids))
	for _, r := range rows {
		if slices.Contains(ids, id(r)) {
			out = append(out, r)
		}
	}
	return out
}
