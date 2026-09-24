package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

type fakeTokenAuth struct {
	tokens map[string]*models.OAuthAccess
}

func (f fakeTokenAuth) Authenticate(_ context.Context, token string) (*models.OAuthAccess, error) {
	a, ok := f.tokens[token]
	if !ok {
		return nil, oauth.ErrInvalidToken
	}
	return a, nil
}
func (fakeTokenAuth) ResourceURL() string { return "https://tb.example.com/mcp" }

type fakeKeys struct {
	repos.APIKeyRepository
	keys []*models.APIKey
}

func (f fakeKeys) List(context.Context) ([]*models.APIKey, error) { return f.keys, nil }

type fakeUsers struct {
	repos.APIUserRepository
	users map[int]*models.APIUser
}

func (f fakeUsers) Get(_ context.Context, id int) (*models.APIUser, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, models.ErrAPIUserNotFound
	}
	return u, nil
}

type fakeWorkflows struct{ ws []*models.Workflow }

func (f fakeWorkflows) List(context.Context) ([]*models.Workflow, error) { return f.ws, nil }
func (f fakeWorkflows) Get(_ context.Context, id int) (*models.Workflow, error) {
	for _, w := range f.ws {
		if w.ID == id {
			return w, nil
		}
	}
	return nil, models.ErrWorkflowNotFound
}

func sampleWorkflow() *models.Workflow {
	rid := 4
	return &models.Workflow{
		ID: 7, Name: "Help Desk", BoardID: 27, BoardName: "Help Desk", Enabled: true,
		Nodes: []models.Node{
			{ID: "t1", Kind: models.NodeTrigger, Title: "New ticket", Enabled: true, Events: []models.TriggerEvent{models.TriggerCreated}, Condition: "priority/name = 'High'"},
			{ID: "i1", Kind: models.NodeIf, Enabled: true, Condition: "contact/id in list 3"},
			{ID: "n1", Kind: models.NodeKind(models.ActionNotify), Title: "Tell the room", Enabled: true,
				ActionSettings: models.ActionSettings{Notify: &models.NotifyAction{Channel: models.ChannelWebexRoom, RecipientID: &rid, Message: "New high ticket {{ticket.id}}"}}},
			{ID: "s1", Kind: models.NodeKind(models.ActionSetStatus), Enabled: false,
				ActionSettings: models.ActionSettings{SetStatus: &models.SetStatusAction{StatusID: 12, StatusName: "Triage"}}},
			{ID: "t2", Kind: models.NodeTrigger, Title: "Updates", Enabled: false, Events: []models.TriggerEvent{models.TriggerUpdated}},
		},
		Edges: []models.Edge{
			{ID: "e1", From: "t1", To: "i1", Port: models.PortOut},
			{ID: "e2", From: "i1", To: "n1", Port: models.PortYes},
			{ID: "e3", From: "i1", To: "s1", Port: models.PortNo},
		},
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-key"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return New(Params{
		OAuth: fakeTokenAuth{tokens: map[string]*models.OAuthAccess{
			"viewer-token": {GrantID: 1, UserID: 1, ClientName: "Claude", Scopes: []string{"read"}, ExpiresAt: time.Now().Add(time.Hour)},
		}},
		Keys: fakeKeys{keys: []*models.APIKey{{ID: 1, UserID: 2, KeyHash: hash}}},
		Users: fakeUsers{users: map[int]*models.APIUser{
			1: {ID: 1, EmailAddress: "viewer@example.com", Role: models.RoleViewer},
			2: {ID: 2, EmailAddress: "admin@example.com", Role: models.RoleAdmin},
		}},
		Deps: Deps{Workflows: fakeWorkflows{ws: []*models.Workflow{sampleWorkflow()}}},
	})
}

type bearerTransport struct{ token string }

func (b bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return http.DefaultTransport.RoundTrip(r)
}

func connect(t *testing.T, url, token string) *sdk.ClientSession {
	t.Helper()
	tr := &sdk.StreamableClientTransport{Endpoint: url, HTTPClient: &http.Client{Transport: bearerTransport{token}}}
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "test", Version: "0"}, nil).Connect(t.Context(), tr, nil)
	if err != nil {
		t.Fatalf("connect as %s: %v", token, err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func toolNames(t *testing.T, cs *sdk.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	slices.Sort(names)
	return names
}

func TestHandlerRequiresBearer(t *testing.T) {
	srv := httptest.NewServer(newTestService(t).Handler())
	t.Cleanup(srv.Close)

	for _, token := range []string{"", "nope"} {
		req, _ := http.NewRequest(http.MethodPost, srv.URL, strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusUnauthorized {
			t.Fatalf("token %q: status %d, want 401", token, res.StatusCode)
		}
		if wa := res.Header.Get("WWW-Authenticate"); !strings.Contains(wa, `resource_metadata="https://tb.example.com/mcp/.well-known/oauth-protected-resource"`) {
			t.Fatalf("WWW-Authenticate = %q", wa)
		}
	}
}

func TestToolsFilteredByRole(t *testing.T) {
	srv := httptest.NewServer(newTestService(t).Handler())
	t.Cleanup(srv.Close)

	viewer := toolNames(t, connect(t, srv.URL, "viewer-token"))
	admin := toolNames(t, connect(t, srv.URL, "admin-key"))

	for _, n := range viewer {
		if !strings.HasPrefix(n, "ticketbot_") {
			t.Errorf("tool %q lacks the ticketbot_ prefix", n)
		}
	}
	if slices.Contains(viewer, "ticketbot_intake_status") || slices.Contains(viewer, "ticketbot_tail_logs") {
		t.Errorf("viewer sees admin tools: %v", viewer)
	}
	if !slices.Contains(admin, "ticketbot_intake_status") || !slices.Contains(admin, "ticketbot_tail_logs") {
		t.Errorf("admin lacks admin tools: %v", admin)
	}
	if len(admin) != len(viewer)+2 {
		t.Errorf("admin %d tools, viewer %d", len(admin), len(viewer))
	}
	for _, want := range []string{"ticketbot_list_workflows", "ticketbot_get_workflow", "ticketbot_list_runs", "ticketbot_get_run", "ticketbot_ticket_history", "ticketbot_lookup_ids", "ticketbot_simulate", "ticketbot_evaluate_condition"} {
		if !slices.Contains(viewer, want) {
			t.Errorf("viewer lacks %s", want)
		}
	}
}

func TestCallWorkflowTools(t *testing.T) {
	srv := httptest.NewServer(newTestService(t).Handler())
	t.Cleanup(srv.Close)
	cs := connect(t, srv.URL, "viewer-token")

	res, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "ticketbot_list_workflows", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	text := res.Content[0].(*sdk.TextContent).Text
	var rows []workflowRow
	if err := json.Unmarshal([]byte(text), &rows); err != nil {
		t.Fatalf("list_workflows not JSON: %v\n%s", err, text)
	}
	if len(rows) != 1 || rows[0].Lanes != 2 || rows[0].Nodes != 5 || rows[0].BoardName != "Help Desk" {
		t.Fatalf("rows: %+v", rows)
	}

	res, err = cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "ticketbot_get_workflow", Arguments: map[string]any{"id": 7}})
	if err != nil {
		t.Fatal(err)
	}
	walk := res.Content[0].(*sdk.TextContent).Text
	for _, want := range []string{
		"Lane 1:", `"New ticket"`, "fires on: created", "only when: priority/name = 'High'",
		"if contact/id in list 3", "yes:", "notify", "webex_room recipient 4", "New high ticket",
		"no:", `set_status`, "Triage", "[disabled: passes through]",
		"Lane 2:", "[disabled: never fires]", "(nothing wired)",
	} {
		if !strings.Contains(walk, want) {
			t.Errorf("walk lacks %q:\n%s", want, walk)
		}
	}

	res, err = cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "ticketbot_get_workflow", Arguments: map[string]any{"id": 7, "raw": true}})
	if err != nil {
		t.Fatal(err)
	}
	var raw models.Workflow
	if err := json.Unmarshal([]byte(res.Content[0].(*sdk.TextContent).Text), &raw); err != nil || len(raw.Edges) != 3 {
		t.Fatalf("raw document: %v", err)
	}

	res, err = cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "ticketbot_get_workflow", Arguments: map[string]any{"id": 99}})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError || !strings.Contains(res.Content[0].(*sdk.TextContent).Text, "not found") {
		t.Fatalf("missing workflow: %+v", res)
	}

	// A viewer calling an admin tool gets an unknown-tool error, not a result.
	if _, err := cs.CallTool(t.Context(), &sdk.CallToolParams{Name: "ticketbot_tail_logs", Arguments: map[string]any{}}); err == nil {
		t.Fatal("viewer called an admin tool")
	}
}

func TestClampLimit(t *testing.T) {
	for in, want := range map[int]int{0: 20, -3: 20, 5: 5, 100: 100, 500: 100} {
		if got := clampLimit(in); got != want {
			t.Errorf("clampLimit(%d) = %d, want %d", in, got, want)
		}
	}
}
