// Package oauth is the authorization server in front of ticketbot's MCP endpoint. It speaks
// the subset MCP clients need: OAuth 2.1 authorization code with PKCE, refresh token rotation,
// dynamic client registration (RFC 7591), token revocation (RFC 7009) and the two metadata
// documents (RFC 8414, RFC 9728). Clients are public; there are no client secrets. Tokens and
// codes are opaque random strings stored as SHA-256, like dashboard sessions.
package oauth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

const (
	// ScopeRead covers every read tool. ScopeWrite is defined so grants have a stable shape, but
	// it is not offered until write tools exist.
	ScopeRead  = "read"
	ScopeWrite = "write"

	// MCPPath is where the MCP endpoint lives under ROOT_URL; it is the protected resource.
	MCPPath = "/mcp"
	// ConsentPath is the dashboard view that shows the consent screen. The authorize endpoint
	// forwards its whole query string there.
	ConsentPath = "/oauth/consent"

	accessTTL       = time.Hour
	refreshTTL      = 30 * 24 * time.Hour
	codeTTL         = 5 * time.Minute
	registrationTTL = 10 * time.Minute
	maxClientName   = 100
)

// OfferedScopes is what a client may ask for and what the consent screen lists.
var OfferedScopes = []string{ScopeRead}

var scopeDescriptions = map[string]string{
	ScopeRead:  "Read workflows, run history, ticket history, lists, forwards and settings",
	ScopeWrite: "Change workflows, lists, forwards and settings",
}

var ErrInvalidToken = errors.New("invalid or expired access token")

// Error is an OAuth protocol error in the wire shape of RFC 6749 §5.2. When RedirectTo is set
// the browser should be sent there (the error is in its query) instead of shown the body.
type Error struct {
	Code        string `json:"error"`
	Description string `json:"error_description,omitempty"`
	Status      int    `json:"-"`
	RedirectTo  string `json:"-"`
}

func (e *Error) Error() string {
	if e.Description == "" {
		return e.Code
	}
	return e.Code + ": " + e.Description
}

func badRequest(code, desc string) *Error {
	return &Error{Code: code, Description: desc, Status: http.StatusBadRequest}
}

type Params struct {
	Repo    repos.OAuthRepository
	Cfg     *models.Config
	RootURL string
	// Now is for tests; nil means time.Now.
	Now func() time.Time
}

type Service struct {
	repo    repos.OAuthRepository
	cfg     *models.Config
	rootURL string
	now     func() time.Time
}

func New(p Params) *Service {
	s := &Service{repo: p.Repo, cfg: p.Cfg, rootURL: strings.TrimRight(p.RootURL, "/"), now: p.Now}
	if s.now == nil {
		s.now = time.Now
	}
	return s
}

// Enabled reports whether the MCP server and these endpoints are switched on. ROOT_URL is
// required because every metadata URL is built from it.
func (s *Service) Enabled() bool { return s.cfg.MCPEnabled && s.rootURL != "" }

// ResourceURL is the identifier of the protected resource: the MCP endpoint.
func (s *Service) ResourceURL() string { return s.rootURL + MCPPath }

// ProtectedResourceMetadata is RFC 9728, served at /.well-known/oauth-protected-resource.
type ProtectedResourceMetadata struct {
	Resource               string   `json:"resource"`
	AuthorizationServers   []string `json:"authorization_servers"`
	ScopesSupported        []string `json:"scopes_supported"`
	BearerMethodsSupported []string `json:"bearer_methods_supported"`
}

func (s *Service) ProtectedResourceMetadata() ProtectedResourceMetadata {
	return ProtectedResourceMetadata{
		Resource:               s.ResourceURL(),
		AuthorizationServers:   []string{s.rootURL},
		ScopesSupported:        slices.Clone(OfferedScopes),
		BearerMethodsSupported: []string{"header"},
	}
}

// ServerMetadata is RFC 8414, served at /.well-known/oauth-authorization-server.
type ServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	ScopesSupported                   []string `json:"scopes_supported"`
}

func (s *Service) ServerMetadata() ServerMetadata {
	return ServerMetadata{
		Issuer:                            s.rootURL,
		AuthorizationEndpoint:             s.rootURL + "/oauth/authorize",
		TokenEndpoint:                     s.rootURL + "/oauth/token",
		RegistrationEndpoint:              s.rootURL + "/oauth/register",
		RevocationEndpoint:                s.rootURL + "/oauth/revoke",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		CodeChallengeMethodsSupported:     []string{"S256"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
		ScopesSupported:                   slices.Clone(OfferedScopes),
	}
}

// RegisterRequest is the RFC 7591 client metadata ticketbot understands. Anything else in the
// body is ignored.
type RegisterRequest struct {
	RedirectURIs            []string `json:"redirect_uris"`
	ClientName              string   `json:"client_name"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// RegisterResponse echoes the accepted metadata with the new client id.
type RegisterResponse struct {
	ClientID                string   `json:"client_id"`
	ClientIDIssuedAt        int64    `json:"client_id_issued_at"`
	ClientName              string   `json:"client_name"`
	RedirectURIs            []string `json:"redirect_uris"`
	TokenEndpointAuthMethod string   `json:"token_endpoint_auth_method"`
	GrantTypes              []string `json:"grant_types"`
	ResponseTypes           []string `json:"response_types"`
}

// Register creates a public client. Redirect URIs must be https, or http on a loopback host
// for command-line clients. The registration expires unless a grant follows.
func (s *Service) Register(ctx context.Context, req RegisterRequest) (*RegisterResponse, error) {
	if len(req.RedirectURIs) == 0 {
		return nil, badRequest("invalid_redirect_uri", "redirect_uris is required")
	}
	for _, u := range req.RedirectURIs {
		if err := validateRedirectURI(u); err != nil {
			return nil, badRequest("invalid_redirect_uri", err.Error())
		}
	}
	if m := req.TokenEndpointAuthMethod; m != "" && m != "none" {
		return nil, badRequest("invalid_client_metadata", "only public clients are supported (token_endpoint_auth_method must be \"none\")")
	}
	grantTypes := req.GrantTypes
	if len(grantTypes) == 0 {
		grantTypes = []string{"authorization_code", "refresh_token"}
	}
	for _, g := range grantTypes {
		if g != "authorization_code" && g != "refresh_token" {
			return nil, badRequest("invalid_client_metadata", fmt.Sprintf("unsupported grant_type %q", g))
		}
	}
	responseTypes := req.ResponseTypes
	if len(responseTypes) == 0 {
		responseTypes = []string{"code"}
	}
	for _, r := range responseTypes {
		if r != "code" {
			return nil, badRequest("invalid_client_metadata", fmt.Sprintf("unsupported response_type %q", r))
		}
	}

	name := strings.TrimSpace(req.ClientName)
	if name == "" {
		name = "Unnamed client"
	}
	if len(name) > maxClientName {
		name = name[:maxClientName]
	}

	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, fmt.Errorf("generating client id: %w", err)
	}
	expires := s.now().Add(registrationTTL)
	c, err := s.repo.CreateClient(ctx, &models.OAuthClient{
		ID: hex.EncodeToString(raw), Name: name, RedirectURIs: req.RedirectURIs, ExpiresAt: &expires,
	})
	if err != nil {
		return nil, fmt.Errorf("storing client: %w", err)
	}
	slog.Info("oauth: client registered", "client_id", c.ID, "name", c.Name, "redirect_uris", c.RedirectURIs)
	return &RegisterResponse{
		ClientID: c.ID, ClientIDIssuedAt: c.CreatedOn.Unix(), ClientName: c.Name, RedirectURIs: c.RedirectURIs,
		TokenEndpointAuthMethod: "none", GrantTypes: grantTypes, ResponseTypes: responseTypes,
	}, nil
}

func validateRedirectURI(raw string) error {
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("%q is not a valid URL", raw)
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("%q must be an absolute URL", raw)
	}
	if u.Fragment != "" {
		return fmt.Errorf("%q must not carry a fragment", raw)
	}
	switch u.Scheme {
	case "https":
		return nil
	case "http":
		if isLoopback(u.Hostname()) {
			return nil
		}
		return fmt.Errorf("%q: http is only allowed on localhost", raw)
	}
	return fmt.Errorf("%q: scheme must be https", raw)
}

func isLoopback(host string) bool {
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	}
	return false
}

// redirectURIMatches applies exact matching, except that a loopback URI may vary its port
// (RFC 8252 §7.3), because command-line clients pick a free one per run.
func redirectURIMatches(registered, requested string) bool {
	if registered == requested {
		return true
	}
	a, errA := url.Parse(registered)
	b, errB := url.Parse(requested)
	if errA != nil || errB != nil || a.Scheme != "http" || b.Scheme != "http" {
		return false
	}
	return isLoopback(a.Hostname()) && a.Hostname() == b.Hostname() && a.Path == b.Path && a.RawQuery == b.RawQuery
}

// AuthorizeRequest is a validated /oauth/authorize query.
type AuthorizeRequest struct {
	ClientID      string
	ClientName    string
	RedirectURI   string
	State         string
	CodeChallenge string
	Scopes        []string
}

// ParseAuthorize validates an authorization request. Errors in the client or redirect URI are
// returned for display (RFC 6749 §4.1.2.1 forbids redirecting to an unverified URI); every
// other error carries RedirectTo so the client learns what went wrong.
func (s *Service) ParseAuthorize(ctx context.Context, q url.Values) (*AuthorizeRequest, error) {
	clientID := q.Get("client_id")
	if clientID == "" {
		return nil, badRequest("invalid_request", "client_id is required")
	}
	client, err := s.repo.GetClient(ctx, clientID)
	if err != nil {
		if errors.Is(err, models.ErrOAuthClientNotFound) {
			return nil, badRequest("invalid_request", "unknown client_id")
		}
		return nil, fmt.Errorf("loading client: %w", err)
	}
	redirectURI := q.Get("redirect_uri")
	if redirectURI == "" {
		if len(client.RedirectURIs) != 1 {
			return nil, badRequest("invalid_request", "redirect_uri is required")
		}
		redirectURI = client.RedirectURIs[0]
	}
	if !slices.ContainsFunc(client.RedirectURIs, func(r string) bool { return redirectURIMatches(r, redirectURI) }) {
		return nil, badRequest("invalid_request", "redirect_uri is not registered for this client")
	}

	req := &AuthorizeRequest{ClientID: client.ID, ClientName: client.Name, RedirectURI: redirectURI, State: q.Get("state"), CodeChallenge: q.Get("code_challenge")}
	fail := func(code, desc string) error {
		return &Error{Code: code, Description: desc, Status: http.StatusFound, RedirectTo: redirectWith(redirectURI, url.Values{"error": {code}, "error_description": {desc}, "state": {req.State}})}
	}

	if q.Get("response_type") != "code" {
		return nil, fail("unsupported_response_type", "response_type must be \"code\"")
	}
	if req.CodeChallenge == "" {
		return nil, fail("invalid_request", "code_challenge is required (PKCE)")
	}
	if m := q.Get("code_challenge_method"); m != "" && m != "S256" {
		return nil, fail("invalid_request", "code_challenge_method must be S256")
	}
	scopes := strings.Fields(q.Get("scope"))
	if len(scopes) == 0 {
		scopes = slices.Clone(OfferedScopes)
	}
	for _, sc := range scopes {
		if !slices.Contains(OfferedScopes, sc) {
			return nil, fail("invalid_scope", fmt.Sprintf("unknown scope %q", sc))
		}
	}
	req.Scopes = slices.Compact(slices.Sorted(slices.Values(scopes)))
	if res := q.Get("resource"); res != "" && !s.isResource(res) {
		return nil, fail("invalid_target", "resource must be "+s.ResourceURL())
	}
	return req, nil
}

func (s *Service) isResource(v string) bool {
	return strings.TrimRight(v, "/") == s.ResourceURL()
}

// ScopeInfo is one line of the consent screen.
type ScopeInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// ConsentInfo is what the consent view renders.
type ConsentInfo struct {
	ClientName   string      `json:"client_name"`
	RedirectHost string      `json:"redirect_host"`
	Scopes       []ScopeInfo `json:"scopes"`
}

// ConsentInfo describes a request for the consent screen. The client name is whatever the
// client registered, so the redirect host is shown beside it as the part it cannot fake.
func (s *Service) ConsentInfo(req *AuthorizeRequest) ConsentInfo {
	info := ConsentInfo{ClientName: req.ClientName}
	if u, err := url.Parse(req.RedirectURI); err == nil {
		info.RedirectHost = u.Host
	}
	for _, sc := range req.Scopes {
		info.Scopes = append(info.Scopes, ScopeInfo{Name: sc, Description: scopeDescriptions[sc]})
	}
	return info
}

// Approve records the grant with the scopes the user ticked (a non-empty subset of those
// requested), mints a code and returns the URL to send the browser to.
func (s *Service) Approve(ctx context.Context, userID int, req *AuthorizeRequest, scopes []string) (string, error) {
	if len(scopes) == 0 {
		return "", badRequest("invalid_scope", "at least one scope must be granted")
	}
	for _, sc := range scopes {
		if !slices.Contains(req.Scopes, sc) {
			return "", badRequest("invalid_scope", fmt.Sprintf("scope %q was not requested", sc))
		}
	}
	scopes = slices.Compact(slices.Sorted(slices.Values(scopes)))

	g, err := s.repo.UpsertGrant(ctx, userID, req.ClientID, scopes)
	if err != nil {
		return "", fmt.Errorf("storing grant: %w", err)
	}
	code, hash, err := newSecret()
	if err != nil {
		return "", err
	}
	if err := s.repo.CreateCode(ctx, &models.OAuthCode{
		CodeHash: hash, GrantID: g.ID, RedirectURI: req.RedirectURI, CodeChallenge: req.CodeChallenge, ExpiresAt: s.now().Add(codeTTL),
	}); err != nil {
		return "", fmt.Errorf("storing code: %w", err)
	}
	slog.Info("oauth: consent granted", "user_id", userID, "client_id", req.ClientID, "scopes", scopes)
	return redirectWith(req.RedirectURI, url.Values{"code": {code}, "state": {req.State}}), nil
}

// Deny returns the URL that tells the client the user said no.
func (s *Service) Deny(req *AuthorizeRequest) string {
	return redirectWith(req.RedirectURI, url.Values{"error": {"access_denied"}, "state": {req.State}})
}

func redirectWith(base string, params url.Values) string {
	u, err := url.Parse(base)
	if err != nil {
		return base
	}
	q := u.Query()
	for k, vs := range params {
		for _, v := range vs {
			if v != "" {
				q.Set(k, v)
			}
		}
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// TokenResponse is RFC 6749 §5.1.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// Token handles the token endpoint: an authorization code exchange verified with PKCE, or a
// refresh. Refresh tokens rotate; presenting one that was already rotated revokes the grant.
func (s *Service) Token(ctx context.Context, form url.Values) (*TokenResponse, error) {
	clientID := form.Get("client_id")
	if clientID == "" {
		return nil, &Error{Code: "invalid_client", Description: "client_id is required", Status: http.StatusUnauthorized}
	}
	if res := form.Get("resource"); res != "" && !s.isResource(res) {
		return nil, badRequest("invalid_target", "resource must be "+s.ResourceURL())
	}

	switch form.Get("grant_type") {
	case "authorization_code":
		return s.exchangeCode(ctx, clientID, form)
	case "refresh_token":
		return s.refresh(ctx, clientID, form)
	case "":
		return nil, badRequest("invalid_request", "grant_type is required")
	default:
		return nil, badRequest("unsupported_grant_type", "grant_type must be authorization_code or refresh_token")
	}
}

func (s *Service) exchangeCode(ctx context.Context, clientID string, form url.Values) (*TokenResponse, error) {
	code, verifier := form.Get("code"), form.Get("code_verifier")
	if code == "" || verifier == "" {
		return nil, badRequest("invalid_request", "code and code_verifier are required")
	}
	c, err := s.repo.TakeCode(ctx, hashSecret(code))
	if err != nil {
		if errors.Is(err, models.ErrOAuthCodeNotFound) {
			return nil, badRequest("invalid_grant", "unknown or already used code")
		}
		return nil, fmt.Errorf("taking code: %w", err)
	}
	if !s.now().Before(c.ExpiresAt) {
		return nil, badRequest("invalid_grant", "code has expired")
	}
	g, err := s.repo.GetGrant(ctx, c.GrantID)
	if err != nil {
		return nil, fmt.Errorf("loading grant: %w", err)
	}
	if g.ClientID != clientID {
		return nil, badRequest("invalid_grant", "code was issued to another client")
	}
	if ru := form.Get("redirect_uri"); ru != "" && ru != c.RedirectURI {
		return nil, badRequest("invalid_grant", "redirect_uri does not match the authorization request")
	}
	if !pkceMatches(c.CodeChallenge, verifier) {
		return nil, badRequest("invalid_grant", "code_verifier does not match code_challenge")
	}
	return s.issue(ctx, g)
}

func (s *Service) refresh(ctx context.Context, clientID string, form url.Values) (*TokenResponse, error) {
	raw := form.Get("refresh_token")
	if raw == "" {
		return nil, badRequest("invalid_request", "refresh_token is required")
	}
	t, err := s.repo.GetToken(ctx, hashSecret(raw))
	if err != nil {
		if errors.Is(err, models.ErrOAuthTokenNotFound) {
			return nil, badRequest("invalid_grant", "unknown refresh token")
		}
		return nil, fmt.Errorf("loading refresh token: %w", err)
	}
	if t.Kind != models.OAuthRefreshToken {
		return nil, badRequest("invalid_grant", "not a refresh token")
	}
	g, err := s.repo.GetGrant(ctx, t.GrantID)
	if err != nil {
		if errors.Is(err, models.ErrOAuthGrantNotFound) {
			return nil, badRequest("invalid_grant", "grant has been revoked")
		}
		return nil, fmt.Errorf("loading grant: %w", err)
	}
	if g.ClientID != clientID {
		return nil, badRequest("invalid_grant", "refresh token was issued to another client")
	}
	if t.UsedAt != nil {
		// Replay of a rotated token: someone holds a copy. Cut off every token on the grant.
		slog.Warn("oauth: rotated refresh token replayed; revoking grant", "grant_id", g.ID, "user_id", g.UserID, "client_id", g.ClientID)
		if err := s.repo.DeleteGrant(ctx, g.ID); err != nil && !errors.Is(err, models.ErrOAuthGrantNotFound) {
			return nil, fmt.Errorf("revoking replayed grant: %w", err)
		}
		return nil, badRequest("invalid_grant", "refresh token has been revoked")
	}
	if !s.now().Before(t.ExpiresAt) {
		return nil, badRequest("invalid_grant", "refresh token has expired")
	}
	for _, sc := range strings.Fields(form.Get("scope")) {
		if !slices.Contains(g.Scopes, sc) {
			return nil, badRequest("invalid_scope", fmt.Sprintf("scope %q was not granted", sc))
		}
	}
	if err := s.repo.MarkTokenUsed(ctx, t.ID); err != nil {
		return nil, fmt.Errorf("rotating refresh token: %w", err)
	}
	return s.issue(ctx, g)
}

func (s *Service) issue(ctx context.Context, g *models.OAuthGrant) (*TokenResponse, error) {
	now := s.now()
	access, accessHash, err := newSecret()
	if err != nil {
		return nil, err
	}
	refresh, refreshHash, err := newSecret()
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthAccessToken, TokenHash: accessHash, ExpiresAt: now.Add(accessTTL)}); err != nil {
		return nil, fmt.Errorf("storing access token: %w", err)
	}
	if _, err := s.repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthRefreshToken, TokenHash: refreshHash, ExpiresAt: now.Add(refreshTTL)}); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}
	return &TokenResponse{
		AccessToken: access, TokenType: "Bearer", ExpiresIn: int(accessTTL / time.Second),
		RefreshToken: refresh, Scope: strings.Join(g.Scopes, " "),
	}, nil
}

// Revoke (RFC 7009) ends the whole grant behind an access or refresh token. Unknown tokens
// are not an error, so the response never reveals whether one existed.
func (s *Service) Revoke(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	t, err := s.repo.GetToken(ctx, hashSecret(raw))
	if err != nil {
		if errors.Is(err, models.ErrOAuthTokenNotFound) {
			return nil
		}
		return fmt.Errorf("loading token: %w", err)
	}
	if err := s.repo.DeleteGrant(ctx, t.GrantID); err != nil && !errors.Is(err, models.ErrOAuthGrantNotFound) {
		return fmt.Errorf("revoking grant: %w", err)
	}
	slog.Info("oauth: grant revoked by client", "grant_id", t.GrantID)
	return nil
}

// Authenticate resolves a bearer access token to its user and scopes, and notes the grant
// was used. Anything but a live access token is ErrInvalidToken.
func (s *Service) Authenticate(ctx context.Context, raw string) (*models.OAuthAccess, error) {
	if raw == "" {
		return nil, ErrInvalidToken
	}
	a, err := s.repo.ResolveAccessToken(ctx, hashSecret(raw))
	if err != nil {
		if errors.Is(err, models.ErrOAuthTokenNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("resolving access token: %w", err)
	}
	if !s.now().Before(a.ExpiresAt) {
		return nil, ErrInvalidToken
	}
	if err := s.repo.TouchGrant(ctx, a.GrantID); err != nil {
		slog.Warn("oauth: recording grant use", "grant_id", a.GrantID, "error", err)
	}
	return a, nil
}

// ListGrants is the Connected apps list for one user.
func (s *Service) ListGrants(ctx context.Context, userID int) ([]*models.OAuthGrant, error) {
	gs, err := s.repo.ListGrantsByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("listing grants: %w", err)
	}
	return gs, nil
}

// RevokeGrant deletes a grant, and with it every token, provided it belongs to userID.
func (s *Service) RevokeGrant(ctx context.Context, userID, grantID int) error {
	g, err := s.repo.GetGrant(ctx, grantID)
	if err != nil {
		return err
	}
	if g.UserID != userID {
		return models.ErrOAuthGrantNotFound
	}
	if err := s.repo.DeleteGrant(ctx, grantID); err != nil {
		return err
	}
	slog.Info("oauth: grant revoked", "grant_id", grantID, "user_id", userID, "client_id", g.ClientID)
	return nil
}

// Purge implements intake.Purger: expired codes and tokens go, and so do registrations that
// never reached a grant.
func (s *Service) Purge(ctx context.Context, now time.Time) {
	c, err := s.repo.DeleteExpired(ctx, now)
	if err != nil {
		slog.Warn("oauth: purging expired rows", "error", err)
		return
	}
	if c.Codes+c.Tokens+c.Clients > 0 {
		slog.Info("oauth: purged expired rows", "codes", c.Codes, "tokens", c.Tokens, "clients", c.Clients)
	}
}

// newSecret returns a 256-bit random token in URL-safe base64 and its storage hash.
func newSecret() (token string, hash []byte, err error) {
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		return "", nil, fmt.Errorf("generating token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(raw)
	return token, hashSecret(token), nil
}

func hashSecret(token string) []byte {
	h := sha256.Sum256([]byte(token))
	return h[:]
}

// pkceMatches checks S256: BASE64URL(SHA256(verifier)) == challenge.
func pkceMatches(challenge, verifier string) bool {
	sum := sha256.Sum256([]byte(verifier))
	got := base64.RawURLEncoding.EncodeToString(sum[:])
	return subtle.ConstantTimeCompare([]byte(got), []byte(challenge)) == 1
}
