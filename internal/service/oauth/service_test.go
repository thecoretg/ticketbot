package oauth

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// fakeRepo is an in-memory repos.OAuthRepository.
type fakeRepo struct {
	clients map[string]*models.OAuthClient
	grants  map[int]*models.OAuthGrant
	codes   map[string]*models.OAuthCode
	tokens  map[string]*models.OAuthToken
	nextID  int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{clients: map[string]*models.OAuthClient{}, grants: map[int]*models.OAuthGrant{}, codes: map[string]*models.OAuthCode{}, tokens: map[string]*models.OAuthToken{}}
}

func (f *fakeRepo) CreateClient(_ context.Context, c *models.OAuthClient) (*models.OAuthClient, error) {
	cp := *c
	cp.CreatedOn = time.Now()
	f.clients[c.ID] = &cp
	return &cp, nil
}

func (f *fakeRepo) GetClient(_ context.Context, id string) (*models.OAuthClient, error) {
	c, ok := f.clients[id]
	if !ok {
		return nil, models.ErrOAuthClientNotFound
	}
	return c, nil
}

func (f *fakeRepo) UpsertGrant(_ context.Context, userID int, clientID string, scopes []string) (*models.OAuthGrant, error) {
	for _, g := range f.grants {
		if g.UserID == userID && g.ClientID == clientID {
			g.Scopes = scopes
			return g, nil
		}
	}
	f.nextID++
	g := &models.OAuthGrant{ID: f.nextID, UserID: userID, ClientID: clientID, Scopes: scopes, CreatedOn: time.Now()}
	f.grants[g.ID] = g
	if c, ok := f.clients[clientID]; ok {
		c.ExpiresAt = nil
	}
	return g, nil
}

func (f *fakeRepo) GetGrant(_ context.Context, id int) (*models.OAuthGrant, error) {
	g, ok := f.grants[id]
	if !ok {
		return nil, models.ErrOAuthGrantNotFound
	}
	return g, nil
}

func (f *fakeRepo) ListGrantsByUser(_ context.Context, userID int) ([]*models.OAuthGrant, error) {
	var out []*models.OAuthGrant
	for _, g := range f.grants {
		if g.UserID == userID {
			out = append(out, g)
		}
	}
	return out, nil
}

func (f *fakeRepo) DeleteGrant(_ context.Context, id int) error {
	if _, ok := f.grants[id]; !ok {
		return models.ErrOAuthGrantNotFound
	}
	delete(f.grants, id)
	for k, t := range f.tokens {
		if t.GrantID == id {
			delete(f.tokens, k)
		}
	}
	for k, c := range f.codes {
		if c.GrantID == id {
			delete(f.codes, k)
		}
	}
	return nil
}

func (f *fakeRepo) TouchGrant(_ context.Context, id int) error {
	now := time.Now()
	f.grants[id].LastUsedAt = &now
	return nil
}

func (f *fakeRepo) CreateCode(_ context.Context, c *models.OAuthCode) error {
	f.codes[string(c.CodeHash)] = c
	return nil
}

func (f *fakeRepo) TakeCode(_ context.Context, hash []byte) (*models.OAuthCode, error) {
	c, ok := f.codes[string(hash)]
	if !ok {
		return nil, models.ErrOAuthCodeNotFound
	}
	delete(f.codes, string(hash))
	return c, nil
}

func (f *fakeRepo) CreateToken(_ context.Context, t *models.OAuthToken) (*models.OAuthToken, error) {
	f.nextID++
	cp := *t
	cp.ID = f.nextID
	f.tokens[string(t.TokenHash)] = &cp
	return &cp, nil
}

func (f *fakeRepo) GetToken(_ context.Context, hash []byte) (*models.OAuthToken, error) {
	t, ok := f.tokens[string(hash)]
	if !ok {
		return nil, models.ErrOAuthTokenNotFound
	}
	return t, nil
}

func (f *fakeRepo) MarkTokenUsed(_ context.Context, id int) error {
	for _, t := range f.tokens {
		if t.ID == id {
			now := time.Now()
			t.UsedAt = &now
		}
	}
	return nil
}

func (f *fakeRepo) ResolveAccessToken(_ context.Context, hash []byte) (*models.OAuthAccess, error) {
	t, ok := f.tokens[string(hash)]
	if !ok || t.Kind != models.OAuthAccessToken {
		return nil, models.ErrOAuthTokenNotFound
	}
	g := f.grants[t.GrantID]
	return &models.OAuthAccess{GrantID: g.ID, UserID: g.UserID, ClientName: f.clients[g.ClientID].Name, Scopes: g.Scopes, ExpiresAt: t.ExpiresAt}, nil
}

func (f *fakeRepo) DeleteExpired(_ context.Context, now time.Time) (models.OAuthPurgeCounts, error) {
	var c models.OAuthPurgeCounts
	for k, v := range f.codes {
		if v.ExpiresAt.Before(now) {
			delete(f.codes, k)
			c.Codes++
		}
	}
	for k, v := range f.tokens {
		if v.ExpiresAt.Before(now) {
			delete(f.tokens, k)
			c.Tokens++
		}
	}
	for k, v := range f.clients {
		if v.ExpiresAt != nil && v.ExpiresAt.Before(now) {
			delete(f.clients, k)
			c.Clients++
		}
	}
	return c, nil
}

const root = "https://tb.example.com"

func newService(t *testing.T) (*Service, *fakeRepo, *time.Time) {
	t.Helper()
	repo := newFakeRepo()
	now := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	svc := New(Params{Repo: repo, Cfg: &models.Config{MCPEnabled: true}, RootURL: root + "/", Now: func() time.Time { return now }})
	return svc, repo, &now
}

func oauthErr(t *testing.T, err error, code string) *Error {
	t.Helper()
	var oe *Error
	if !errors.As(err, &oe) {
		t.Fatalf("want *Error %s, got %v", code, err)
	}
	if oe.Code != code {
		t.Fatalf("want error %s, got %s (%s)", code, oe.Code, oe.Description)
	}
	return oe
}

func pkce(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func register(t *testing.T, svc *Service, uris ...string) *RegisterResponse {
	t.Helper()
	res, err := svc.Register(context.Background(), RegisterRequest{RedirectURIs: uris, ClientName: "Claude"})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func TestEnabledNeedsRootURL(t *testing.T) {
	cfg := &models.Config{MCPEnabled: true}
	if New(Params{Cfg: cfg}).Enabled() {
		t.Fatal("enabled without ROOT_URL")
	}
	if !New(Params{Cfg: cfg, RootURL: root}).Enabled() {
		t.Fatal("not enabled with ROOT_URL")
	}
	cfg.MCPEnabled = false
	if New(Params{Cfg: cfg, RootURL: root}).Enabled() {
		t.Fatal("enabled with the switch off")
	}
}

func TestMetadata(t *testing.T) {
	svc, _, _ := newService(t)
	pr := svc.ProtectedResourceMetadata()
	if pr.Resource != root+"/mcp" || pr.AuthorizationServers[0] != root {
		t.Fatalf("protected resource metadata: %+v", pr)
	}
	sm := svc.ServerMetadata()
	if sm.Issuer != root || sm.TokenEndpoint != root+"/oauth/token" || sm.RegistrationEndpoint != root+"/oauth/register" {
		t.Fatalf("server metadata: %+v", sm)
	}
	if !slices.Equal(sm.CodeChallengeMethodsSupported, []string{"S256"}) || !slices.Equal(sm.TokenEndpointAuthMethodsSupported, []string{"none"}) {
		t.Fatalf("server metadata: %+v", sm)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, repo, now := newService(t)
	tests := []struct {
		name string
		req  RegisterRequest
		code string
	}{
		{"no redirect uris", RegisterRequest{}, "invalid_redirect_uri"},
		{"plain http", RegisterRequest{RedirectURIs: []string{"http://example.com/cb"}}, "invalid_redirect_uri"},
		{"relative", RegisterRequest{RedirectURIs: []string{"/cb"}}, "invalid_redirect_uri"},
		{"fragment", RegisterRequest{RedirectURIs: []string{"https://example.com/cb#x"}}, "invalid_redirect_uri"},
		{"custom scheme", RegisterRequest{RedirectURIs: []string{"myapp://cb"}}, "invalid_redirect_uri"},
		{"secret client", RegisterRequest{RedirectURIs: []string{"https://claude.ai/cb"}, TokenEndpointAuthMethod: "client_secret_basic"}, "invalid_client_metadata"},
		{"implicit", RegisterRequest{RedirectURIs: []string{"https://claude.ai/cb"}, ResponseTypes: []string{"token"}}, "invalid_client_metadata"},
		{"client credentials", RegisterRequest{RedirectURIs: []string{"https://claude.ai/cb"}, GrantTypes: []string{"client_credentials"}}, "invalid_client_metadata"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Register(context.Background(), tt.req)
			oauthErr(t, err, tt.code)
		})
	}

	res, err := svc.Register(context.Background(), RegisterRequest{
		RedirectURIs: []string{"https://claude.ai/api/mcp/auth_callback", "http://localhost:3334/callback"},
		ClientName:   "  Claude  ", TokenEndpointAuthMethod: "none",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.ClientName != "Claude" || res.TokenEndpointAuthMethod != "none" || len(res.ClientID) != 32 {
		t.Fatalf("register response: %+v", res)
	}
	if !slices.Equal(res.GrantTypes, []string{"authorization_code", "refresh_token"}) || !slices.Equal(res.ResponseTypes, []string{"code"}) {
		t.Fatalf("defaults not applied: %+v", res)
	}
	c := repo.clients[res.ClientID]
	if c.ExpiresAt == nil || !c.ExpiresAt.Equal(now.Add(registrationTTL)) {
		t.Fatalf("registration expiry: %v", c.ExpiresAt)
	}
	unnamed, err := svc.Register(context.Background(), RegisterRequest{RedirectURIs: []string{"https://claude.ai/cb"}})
	if err != nil {
		t.Fatal(err)
	}
	if unnamed.ClientName != "Unnamed client" {
		t.Fatalf("default name: %q", unnamed.ClientName)
	}
}

func authorizeQuery(clientID, redirect, challenge string) url.Values {
	return url.Values{
		"response_type": {"code"}, "client_id": {clientID}, "redirect_uri": {redirect},
		"code_challenge": {challenge}, "code_challenge_method": {"S256"}, "state": {"xyz"},
	}
}

func TestParseAuthorize(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	c := register(t, svc, "https://claude.ai/cb", "http://127.0.0.1:1/callback")
	challenge := pkce("verifier-verifier-verifier-verifier-verifier")

	t.Run("unknown client is not redirected", func(t *testing.T) {
		q := authorizeQuery("nope", "https://claude.ai/cb", challenge)
		oe := oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_request")
		if oe.RedirectTo != "" {
			t.Fatal("redirected to an unverified client")
		}
	})
	t.Run("unregistered redirect is not redirected", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://evil.example/cb", challenge)
		oe := oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_request")
		if oe.RedirectTo != "" {
			t.Fatal("redirected to an unregistered uri")
		}
	})
	t.Run("missing pkce redirects with error", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", "")
		oe := oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_request")
		u, _ := url.Parse(oe.RedirectTo)
		if u.Host != "claude.ai" || u.Query().Get("error") != "invalid_request" || u.Query().Get("state") != "xyz" {
			t.Fatalf("redirect: %s", oe.RedirectTo)
		}
	})
	t.Run("plain method rejected", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("code_challenge_method", "plain")
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_request")
	})
	t.Run("wrong response type", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("response_type", "token")
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "unsupported_response_type")
	})
	t.Run("unknown scope", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("scope", "read admin")
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_scope")
	})
	t.Run("write not offered yet", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("scope", "write")
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_scope")
	})
	t.Run("wrong resource", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("resource", "https://other.example/mcp")
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_target")
	})
	t.Run("defaults scope and accepts resource", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "https://claude.ai/cb", challenge)
		q.Set("resource", root+"/mcp/")
		req, err := svc.ParseAuthorize(ctx, q)
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(req.Scopes, []string{"read"}) || req.ClientName != "Claude" || req.State != "xyz" {
			t.Fatalf("request: %+v", req)
		}
		info := svc.ConsentInfo(req)
		if info.RedirectHost != "claude.ai" || len(info.Scopes) != 1 || info.Scopes[0].Description == "" {
			t.Fatalf("consent info: %+v", info)
		}
	})
	t.Run("loopback port may vary", func(t *testing.T) {
		q := authorizeQuery(c.ClientID, "http://127.0.0.1:51234/callback", challenge)
		if _, err := svc.ParseAuthorize(ctx, q); err != nil {
			t.Fatal(err)
		}
		q = authorizeQuery(c.ClientID, "http://127.0.0.1:51234/other", challenge)
		oauthErr(t, mustErr(svc.ParseAuthorize(ctx, q)), "invalid_request")
	})
}

func mustErr[T any](_ T, err error) error { return err }

// approve runs the browser half of the flow and returns the code from the redirect.
func approve(t *testing.T, svc *Service, clientID, redirect, challenge string, userID int) string {
	t.Helper()
	ctx := context.Background()
	req, err := svc.ParseAuthorize(ctx, authorizeQuery(clientID, redirect, challenge))
	if err != nil {
		t.Fatal(err)
	}
	target, err := svc.Approve(ctx, userID, req, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(target)
	if err != nil {
		t.Fatal(err)
	}
	if u.Query().Get("state") != "xyz" || u.Query().Get("code") == "" {
		t.Fatalf("approve redirect: %s", target)
	}
	return u.Query().Get("code")
}

func TestCodeFlow(t *testing.T) {
	svc, repo, now := newService(t)
	ctx := context.Background()
	c := register(t, svc, "https://claude.ai/cb")
	verifier := "a-long-enough-verifier-string-for-pkce-testing-1"
	code := approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 7)

	if repo.clients[c.ClientID].ExpiresAt != nil {
		t.Fatal("grant did not keep the client")
	}

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {c.ClientID}, "redirect_uri": {"https://claude.ai/cb"}}

	t.Run("wrong verifier burns the code", func(t *testing.T) {
		f := cloneValues(form)
		f.Set("code_verifier", "wrong")
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_grant")
		f.Set("code_verifier", verifier)
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_grant")
	})

	code = approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 7)
	form.Set("code", code)
	form.Set("code_verifier", verifier)

	t.Run("other client cannot redeem", func(t *testing.T) {
		other := register(t, svc, "https://claude.ai/cb")
		f := cloneValues(form)
		f.Set("client_id", other.ClientID)
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_grant")
	})

	code = approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 7)
	form.Set("code", code)

	t.Run("expired code", func(t *testing.T) {
		*now = now.Add(codeTTL + time.Second)
		oauthErr(t, mustErr(svc.Token(ctx, form)), "invalid_grant")
	})

	code = approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 7)
	form.Set("code", code)
	res, err := svc.Token(ctx, form)
	if err != nil {
		t.Fatal(err)
	}
	if res.TokenType != "Bearer" || res.ExpiresIn != 3600 || res.Scope != "read" || res.AccessToken == "" || res.RefreshToken == "" {
		t.Fatalf("token response: %+v", res)
	}

	t.Run("code is one-shot", func(t *testing.T) {
		oauthErr(t, mustErr(svc.Token(ctx, form)), "invalid_grant")
	})

	t.Run("access token authenticates and expires", func(t *testing.T) {
		a, err := svc.Authenticate(ctx, res.AccessToken)
		if err != nil {
			t.Fatal(err)
		}
		if a.UserID != 7 || !slices.Equal(a.Scopes, []string{"read"}) {
			t.Fatalf("access: %+v", a)
		}
		if repo.grants[a.GrantID].LastUsedAt == nil {
			t.Fatal("grant use not recorded")
		}
		if _, err := svc.Authenticate(ctx, res.RefreshToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("refresh token accepted as access token: %v", err)
		}
		*now = now.Add(accessTTL + time.Second)
		if _, err := svc.Authenticate(ctx, res.AccessToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("expired access token accepted: %v", err)
		}
	})

	t.Run("refresh rotates and replay revokes", func(t *testing.T) {
		rf := url.Values{"grant_type": {"refresh_token"}, "refresh_token": {res.RefreshToken}, "client_id": {c.ClientID}}
		next, err := svc.Token(ctx, rf)
		if err != nil {
			t.Fatal(err)
		}
		if next.RefreshToken == res.RefreshToken || next.AccessToken == res.AccessToken {
			t.Fatal("tokens were not rotated")
		}
		if _, err := svc.Authenticate(ctx, next.AccessToken); err != nil {
			t.Fatal(err)
		}
		// Replay the old refresh token: the whole grant goes, including the new access token.
		oauthErr(t, mustErr(svc.Token(ctx, rf)), "invalid_grant")
		if _, err := svc.Authenticate(ctx, next.AccessToken); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("access token survived replay revocation: %v", err)
		}
		rf.Set("refresh_token", next.RefreshToken)
		oauthErr(t, mustErr(svc.Token(ctx, rf)), "invalid_grant")
		if len(repo.grants) != 0 {
			t.Fatalf("grant survived: %d", len(repo.grants))
		}
	})
}

func TestRefreshValidation(t *testing.T) {
	svc, _, now := newService(t)
	ctx := context.Background()
	c := register(t, svc, "https://claude.ai/cb")
	verifier := "a-long-enough-verifier-string-for-pkce-testing-2"
	code := approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 1)
	res, err := svc.Token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {c.ClientID}, "code_verifier": {verifier}})
	if err != nil {
		t.Fatal(err)
	}

	rf := func() url.Values {
		return url.Values{"grant_type": {"refresh_token"}, "refresh_token": {res.RefreshToken}, "client_id": {c.ClientID}}
	}
	t.Run("other client", func(t *testing.T) {
		f := rf()
		f.Set("client_id", register(t, svc, "https://claude.ai/cb").ClientID)
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_grant")
	})
	t.Run("access token is not a refresh token", func(t *testing.T) {
		f := rf()
		f.Set("refresh_token", res.AccessToken)
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_grant")
	})
	t.Run("scope must be granted", func(t *testing.T) {
		f := rf()
		f.Set("scope", "write")
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_scope")
	})
	t.Run("missing client", func(t *testing.T) {
		f := rf()
		f.Del("client_id")
		oauthErr(t, mustErr(svc.Token(ctx, f)), "invalid_client")
	})
	t.Run("unsupported grant type", func(t *testing.T) {
		f := rf()
		f.Set("grant_type", "password")
		oauthErr(t, mustErr(svc.Token(ctx, f)), "unsupported_grant_type")
	})
	t.Run("expired", func(t *testing.T) {
		*now = now.Add(refreshTTL + time.Second)
		oauthErr(t, mustErr(svc.Token(ctx, rf())), "invalid_grant")
	})
}

func TestRevokeAndGrants(t *testing.T) {
	svc, repo, _ := newService(t)
	ctx := context.Background()
	c := register(t, svc, "https://claude.ai/cb")
	verifier := "a-long-enough-verifier-string-for-pkce-testing-3"
	code := approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 3)
	res, err := svc.Token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {c.ClientID}, "code_verifier": {verifier}})
	if err != nil {
		t.Fatal(err)
	}

	gs, err := svc.ListGrants(ctx, 3)
	if err != nil || len(gs) != 1 {
		t.Fatalf("grants: %v %v", gs, err)
	}
	if err := svc.RevokeGrant(ctx, 4, gs[0].ID); !errors.Is(err, models.ErrOAuthGrantNotFound) {
		t.Fatalf("another user revoked the grant: %v", err)
	}
	if err := svc.Revoke(ctx, "not-a-token"); err != nil {
		t.Fatal(err)
	}
	if err := svc.Revoke(ctx, res.AccessToken); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Authenticate(ctx, res.AccessToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("access token survived revocation: %v", err)
	}
	if len(repo.grants) != 0 {
		t.Fatal("grant survived revocation")
	}

	// Re-approving the same client replaces rather than duplicates the grant.
	code = approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 3)
	_ = code
	code = approve(t, svc, c.ClientID, "https://claude.ai/cb", pkce(verifier), 3)
	_ = code
	if gs, _ = svc.ListGrants(ctx, 3); len(gs) != 1 {
		t.Fatalf("duplicate grants: %d", len(gs))
	}
	if err := svc.RevokeGrant(ctx, 3, gs[0].ID); err != nil {
		t.Fatal(err)
	}
	if gs, _ = svc.ListGrants(ctx, 3); len(gs) != 0 {
		t.Fatal("grant not revoked by owner")
	}
}

func TestApproveScopes(t *testing.T) {
	svc, _, _ := newService(t)
	ctx := context.Background()
	c := register(t, svc, "https://claude.ai/cb")
	req, err := svc.ParseAuthorize(ctx, authorizeQuery(c.ClientID, "https://claude.ai/cb", pkce("v")))
	if err != nil {
		t.Fatal(err)
	}
	oauthErr(t, mustErr(svc.Approve(ctx, 1, req, nil)), "invalid_scope")
	oauthErr(t, mustErr(svc.Approve(ctx, 1, req, []string{"write"})), "invalid_scope")
	deny := svc.Deny(req)
	if !strings.Contains(deny, "error=access_denied") || !strings.Contains(deny, "state=xyz") {
		t.Fatalf("deny redirect: %s", deny)
	}
}

func TestPurge(t *testing.T) {
	svc, repo, now := newService(t)
	ctx := context.Background()
	abandoned := register(t, svc, "https://claude.ai/cb")
	kept := register(t, svc, "https://claude.ai/cb")
	verifier := "a-long-enough-verifier-string-for-pkce-testing-4"
	approve(t, svc, kept.ClientID, "https://claude.ai/cb", pkce(verifier), 1) // code left unexchanged
	code := approve(t, svc, kept.ClientID, "https://claude.ai/cb", pkce(verifier), 1)
	if _, err := svc.Token(ctx, url.Values{"grant_type": {"authorization_code"}, "code": {code}, "client_id": {kept.ClientID}, "code_verifier": {verifier}}); err != nil {
		t.Fatal(err)
	}

	svc.Purge(ctx, *now)
	if len(repo.clients) != 2 || len(repo.codes) != 1 || len(repo.tokens) != 2 {
		t.Fatalf("purge removed live rows: clients=%d codes=%d tokens=%d", len(repo.clients), len(repo.codes), len(repo.tokens))
	}
	svc.Purge(ctx, now.Add(registrationTTL+time.Second))
	if _, ok := repo.clients[abandoned.ClientID]; ok {
		t.Fatal("abandoned client survived")
	}
	if _, ok := repo.clients[kept.ClientID]; !ok {
		t.Fatal("granted client purged")
	}
	if len(repo.codes) != 0 {
		t.Fatal("expired code survived")
	}
	svc.Purge(ctx, now.Add(accessTTL+time.Second))
	if len(repo.tokens) != 1 {
		t.Fatalf("expired access token survived: %d", len(repo.tokens))
	}
}

func cloneValues(v url.Values) url.Values {
	out := url.Values{}
	for k, vs := range v {
		out[k] = slices.Clone(vs)
	}
	return out
}
