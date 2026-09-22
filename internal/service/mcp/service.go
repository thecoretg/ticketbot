// Package mcp serves ticketbot's tools to MCP clients over Streamable HTTP at /mcp. The server
// is stateless: every request is authenticated on its own (an OAuth access token from
// internal/service/oauth, or a dashboard API key) and gets a Server holding only the tools the
// caller may use, so tools/list never shows a viewer an admin tool. Tool names carry a
// ticketbot_ prefix so they cannot be confused with a ConnectWise connector's cw_* tools.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/service/oauth"
	"github.com/thecoretg/ticketbot/models"
	"golang.org/x/crypto/bcrypt"
)

const (
	serverName = "ticketbot"
	// toolPrefix keeps the tool names distinct from the ConnectWise PSA connector's cw_* tools.
	toolPrefix = "ticketbot_"

	defaultLimit = 20
	maxLimit     = 100

	instructions = "ticketbot watches ConnectWise PSA ticket webhooks, runs workflows (lanes of " +
		"trigger, if and action nodes) against them and notifies Webex rooms and people. These " +
		"tools read ticketbot's own state: its workflows, what each run did, the history it kept " +
		"per ticket, its lists, forwards and settings. They are not a ConnectWise client: for " +
		"live ticket data use the ConnectWise PSA connector (cw_* tools). ConnectWise ids are the " +
		"same in both; ticketbot_lookup_ids translates between ids and names."
)

// Principal is who is calling: the dashboard user and the scopes their credential carries. An
// API key carries every scope because it already grants the whole dashboard API.
type Principal struct {
	User   *models.APIUser
	Scopes []string
	// Client is the OAuth client name, or "api-key".
	Client string
}

// permitted reports whether the principal may use a tool needing the role floor and scope.
func (p Principal) permitted(minRole models.Role, scope string) bool {
	return p.User.Role.AtLeast(minRole) && slices.Contains(p.Scopes, scope)
}

type principalKey struct{}

// TokenAuth is the slice of oauth.Service the endpoint needs.
type TokenAuth interface {
	Authenticate(ctx context.Context, token string) (*models.OAuthAccess, error)
	ResourceURL() string
}

// Params are the services the tools read from. Every field is an interface defined in this
// package (see tools.go) so tests can fake the slice they need.
type Params struct {
	OAuth TokenAuth
	Keys  repos.APIKeyRepository
	Users repos.APIUserRepository
	Deps  Deps
}

type Service struct {
	oauth TokenAuth
	keys  repos.APIKeyRepository
	users repos.APIUserRepository
	deps  Deps
	tools []toolDef
}

func New(p Params) *Service {
	s := &Service{oauth: p.OAuth, keys: p.Keys, users: p.Users, deps: p.Deps}
	s.tools = s.toolDefs()
	return s
}

// Handler is the /mcp endpoint: bearer auth, then a stateless Streamable HTTP server built for
// the caller. The 401 carries the resource metadata URL so clients discover the OAuth server.
func (s *Service) Handler() http.Handler {
	// JSONResponse: tools answer quickly and never stream, so plain JSON bodies beat SSE framing.
	h := sdk.NewStreamableHTTPHandler(s.serverFor, &sdk.StreamableHTTPOptions{Stateless: true, JSONResponse: true})
	return auth.RequireBearerToken(s.verify, &auth.RequireBearerTokenOptions{
		ResourceMetadataURL: s.oauth.ResourceURL() + "/.well-known/oauth-protected-resource",
		Scopes:              []string{oauth.ScopeRead},
	})(h)
}

// verify is the auth.TokenVerifier: an OAuth access token first, then an API key.
func (s *Service) verify(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
	if a, err := s.oauth.Authenticate(ctx, token); err == nil {
		u, err := s.users.Get(ctx, a.UserID)
		if err != nil {
			return nil, fmt.Errorf("%w: user", auth.ErrInvalidToken)
		}
		return tokenInfo(&Principal{User: u, Scopes: a.Scopes, Client: a.ClientName}, a.ExpiresAt), nil
	} else if !errors.Is(err, oauth.ErrInvalidToken) {
		return nil, err
	}

	keys, err := s.keys.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("listing api keys: %w", err)
	}
	for _, k := range keys {
		if bcrypt.CompareHashAndPassword(k.KeyHash, []byte(token)) == nil {
			u, err := s.users.Get(ctx, k.UserID)
			if err != nil {
				return nil, fmt.Errorf("%w: user", auth.ErrInvalidToken)
			}
			// API keys do not expire; the SDK insists on an expiry unless told otherwise, so
			// give it a far one rather than relaxing the check for OAuth tokens too.
			return tokenInfo(&Principal{User: u, Scopes: []string{oauth.ScopeRead, oauth.ScopeWrite}, Client: "api-key"}, time.Now().Add(24*time.Hour)), nil
		}
	}
	return nil, auth.ErrInvalidToken
}

func tokenInfo(p *Principal, exp time.Time) *auth.TokenInfo {
	return &auth.TokenInfo{
		Scopes:     p.Scopes,
		Expiration: exp,
		UserID:     fmt.Sprint(p.User.ID),
		Extra:      map[string]any{"principal": p},
	}
}

// principalFrom recovers the Principal RequireBearerToken stored on the request.
func principalFrom(r *http.Request) *Principal {
	ti := auth.TokenInfoFromContext(r.Context())
	if ti == nil {
		return nil
	}
	p, _ := ti.Extra["principal"].(*Principal)
	return p
}

// serverFor builds the per-request Server with the tools this caller may use.
func (s *Service) serverFor(r *http.Request) *sdk.Server {
	p := principalFrom(r)
	if p == nil {
		return nil
	}
	srv := sdk.NewServer(&sdk.Implementation{Name: serverName, Title: "Ticketbot", Version: "2"}, &sdk.ServerOptions{Instructions: instructions})
	srv.AddReceivingMiddleware(s.logCalls(p))
	for _, t := range s.tools {
		if p.permitted(t.minRole, t.scope) {
			t.add(srv)
		}
	}
	return srv
}

// logCalls writes the one structured line per tool call the audit trail consists of.
func (s *Service) logCalls(p *Principal) sdk.Middleware {
	return func(next sdk.MethodHandler) sdk.MethodHandler {
		return func(ctx context.Context, method string, req sdk.Request) (sdk.Result, error) {
			if method != "tools/call" {
				return next(ctx, method, req)
			}
			ctp, _ := req.GetParams().(*sdk.CallToolParamsRaw)
			start := time.Now()
			res, err := next(ctx, method, req)
			attrs := []any{"user_id", p.User.ID, "user", p.User.EmailAddress, "client", p.Client, "duration_ms", time.Since(start).Milliseconds()}
			if ctp != nil {
				attrs = append(attrs, "tool", ctp.Name, "args", summarizeArgs(ctp.Arguments))
			}
			if err != nil {
				attrs = append(attrs, "error", err)
			} else if r, ok := res.(*sdk.CallToolResult); ok && r.IsError {
				attrs = append(attrs, "tool_error", true)
			}
			slog.Info("mcp: tool call", attrs...)
			return res, err
		}
	}
}

// summarizeArgs keeps the audit line short: raw argument JSON, capped.
func summarizeArgs(args json.RawMessage) string {
	s := strings.TrimSpace(string(args))
	if len(s) > 300 {
		s = s[:300] + "…"
	}
	return s
}

// clampLimit applies the default and ceiling for list tools.
func clampLimit(n int) int {
	if n <= 0 {
		return defaultLimit
	}
	return min(n, maxLimit)
}

// textResult is a plain-text tool result.
func textResult(s string) *sdk.CallToolResult {
	return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: s}}}
}
