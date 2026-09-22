package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/thecoretg/ticketbot/models"
)

// Fixtures expire in the year 2000 so DeleteExpired with a 2001 clock only ever touches them.
var (
	y2000 = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	y2001 = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)
	far   = time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
)

func TestOAuthRepoFlow(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOAuthRepo(pool)
	users := NewAPIUserRepo(pool)

	u, err := users.Insert(ctx, "oauth-repo-test@example.com", models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM api_user WHERE id = $1`, u.ID) })

	exp := y2000
	c, err := repo.CreateClient(ctx, &models.OAuthClient{ID: "oauth-test-client", Name: "Test", RedirectURIs: []string{"https://claude.ai/cb", "http://localhost:1/cb"}, ExpiresAt: &exp})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM oauth_client WHERE id LIKE 'oauth-test-%'`) })
	if got, err := repo.GetClient(ctx, c.ID); err != nil || len(got.RedirectURIs) != 2 || got.ExpiresAt == nil {
		t.Fatalf("GetClient: %+v %v", got, err)
	}
	if _, err := repo.GetClient(ctx, "oauth-test-missing"); !errors.Is(err, models.ErrOAuthClientNotFound) {
		t.Fatalf("missing client: %v", err)
	}

	g, err := repo.UpsertGrant(ctx, u.ID, c.ID, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	g2, err := repo.UpsertGrant(ctx, u.ID, c.ID, []string{"read", "write"})
	if err != nil {
		t.Fatal(err)
	}
	if g2.ID != g.ID || len(g2.Scopes) != 2 {
		t.Fatalf("upsert did not replace: %+v then %+v", g, g2)
	}
	if got, _ := repo.GetClient(ctx, c.ID); got.ExpiresAt != nil {
		t.Fatal("grant did not clear the client's registration expiry")
	}
	gs, err := repo.ListGrantsByUser(ctx, u.ID)
	if err != nil || len(gs) != 1 || gs[0].ClientName != "Test" {
		t.Fatalf("ListGrantsByUser: %+v %v", gs, err)
	}

	code := &models.OAuthCode{CodeHash: []byte("oauth-test-code-hash"), GrantID: g.ID, RedirectURI: "https://claude.ai/cb", CodeChallenge: "ch", ExpiresAt: far}
	if err := repo.CreateCode(ctx, code); err != nil {
		t.Fatal(err)
	}
	taken, err := repo.TakeCode(ctx, code.CodeHash)
	if err != nil || taken.GrantID != g.ID || taken.CodeChallenge != "ch" {
		t.Fatalf("TakeCode: %+v %v", taken, err)
	}
	if _, err := repo.TakeCode(ctx, code.CodeHash); !errors.Is(err, models.ErrOAuthCodeNotFound) {
		t.Fatalf("code taken twice: %v", err)
	}

	access, err := repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthAccessToken, TokenHash: []byte("oauth-test-access"), ExpiresAt: far})
	if err != nil {
		t.Fatal(err)
	}
	refresh, err := repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthRefreshToken, TokenHash: []byte("oauth-test-refresh"), ExpiresAt: far})
	if err != nil {
		t.Fatal(err)
	}
	a, err := repo.ResolveAccessToken(ctx, access.TokenHash)
	if err != nil || a.UserID != u.ID || a.GrantID != g.ID || len(a.Scopes) != 2 {
		t.Fatalf("ResolveAccessToken: %+v %v", a, err)
	}
	if _, err := repo.ResolveAccessToken(ctx, refresh.TokenHash); !errors.Is(err, models.ErrOAuthTokenNotFound) {
		t.Fatalf("refresh token resolved as access: %v", err)
	}
	if err := repo.MarkTokenUsed(ctx, refresh.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := repo.GetToken(ctx, refresh.TokenHash); err != nil || got.UsedAt == nil || got.Kind != models.OAuthRefreshToken {
		t.Fatalf("GetToken after MarkTokenUsed: %+v %v", got, err)
	}
	if err := repo.TouchGrant(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := repo.GetGrant(ctx, g.ID); got.LastUsedAt == nil {
		t.Fatal("TouchGrant did not set last_used_at")
	}

	// Deleting the grant cascades to its tokens.
	if err := repo.DeleteGrant(ctx, g.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.DeleteGrant(ctx, g.ID); !errors.Is(err, models.ErrOAuthGrantNotFound) {
		t.Fatalf("second delete: %v", err)
	}
	if _, err := repo.GetToken(ctx, access.TokenHash); !errors.Is(err, models.ErrOAuthTokenNotFound) {
		t.Fatalf("token survived grant delete: %v", err)
	}
}

func TestOAuthRepoDeleteExpired(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewOAuthRepo(pool)
	users := NewAPIUserRepo(pool)

	u, err := users.Insert(ctx, "oauth-purge-test@example.com", models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM api_user WHERE id = $1`, u.ID) })
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM oauth_client WHERE id LIKE 'oauth-test-%'`) })

	old := y2000
	if _, err := repo.CreateClient(ctx, &models.OAuthClient{ID: "oauth-test-abandoned", Name: "a", RedirectURIs: []string{"https://x/cb"}, ExpiresAt: &old}); err != nil {
		t.Fatal(err)
	}
	kept, err := repo.CreateClient(ctx, &models.OAuthClient{ID: "oauth-test-kept", Name: "k", RedirectURIs: []string{"https://x/cb"}, ExpiresAt: &old})
	if err != nil {
		t.Fatal(err)
	}
	g, err := repo.UpsertGrant(ctx, u.ID, kept.ID, []string{"read"})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateCode(ctx, &models.OAuthCode{CodeHash: []byte("oauth-test-old-code"), GrantID: g.ID, RedirectURI: "https://x/cb", CodeChallenge: "c", ExpiresAt: y2000}); err != nil {
		t.Fatal(err)
	}
	if err := repo.CreateCode(ctx, &models.OAuthCode{CodeHash: []byte("oauth-test-live-code"), GrantID: g.ID, RedirectURI: "https://x/cb", CodeChallenge: "c", ExpiresAt: far}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthAccessToken, TokenHash: []byte("oauth-test-old-token"), ExpiresAt: y2000}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CreateToken(ctx, &models.OAuthToken{GrantID: g.ID, Kind: models.OAuthAccessToken, TokenHash: []byte("oauth-test-live-token"), ExpiresAt: far}); err != nil {
		t.Fatal(err)
	}

	counts, err := repo.DeleteExpired(ctx, y2001)
	if err != nil {
		t.Fatal(err)
	}
	if counts.Codes < 1 || counts.Tokens < 1 || counts.Clients < 1 {
		t.Fatalf("counts: %+v", counts)
	}
	if _, err := repo.GetClient(ctx, "oauth-test-abandoned"); !errors.Is(err, models.ErrOAuthClientNotFound) {
		t.Fatalf("abandoned client survived: %v", err)
	}
	if _, err := repo.GetClient(ctx, kept.ID); err != nil {
		t.Fatalf("granted client purged: %v", err)
	}
	if _, err := repo.TakeCode(ctx, []byte("oauth-test-old-code")); !errors.Is(err, models.ErrOAuthCodeNotFound) {
		t.Fatal("expired code survived")
	}
	if _, err := repo.TakeCode(ctx, []byte("oauth-test-live-code")); err != nil {
		t.Fatalf("live code purged: %v", err)
	}
	if _, err := repo.GetToken(ctx, []byte("oauth-test-old-token")); !errors.Is(err, models.ErrOAuthTokenNotFound) {
		t.Fatal("expired token survived")
	}
	if _, err := repo.GetToken(ctx, []byte("oauth-test-live-token")); err != nil {
		t.Fatalf("live token purged: %v", err)
	}
}
