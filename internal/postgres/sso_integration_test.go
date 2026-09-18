package postgres

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/models"
)

func TestSSOStoreFlowState(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewSSOStore(pool)

	fs := entra.FlowState{State: "st", Nonce: "nc", Verifier: "vf", Next: "/panel/#tickets", ExpiresAt: time.Now().Add(time.Minute).Round(time.Millisecond)}
	if err := store.PutState(ctx, "flow-key-1", fs); err != nil {
		t.Fatal(err)
	}
	got, ok, err := store.TakeState(ctx, "flow-key-1")
	if err != nil || !ok {
		t.Fatalf("TakeState: ok=%v err=%v", ok, err)
	}
	if got.State != fs.State || got.Nonce != fs.Nonce || got.Verifier != fs.Verifier || got.Next != fs.Next || !got.ExpiresAt.Equal(fs.ExpiresAt) {
		t.Fatalf("got %+v, want %+v", got, fs)
	}
	// One-shot: a replayed callback finds nothing.
	if _, ok, err := store.TakeState(ctx, "flow-key-1"); err != nil || ok {
		t.Fatalf("second TakeState: ok=%v err=%v", ok, err)
	}
	// Expired rows are swept on the next Put.
	_ = store.PutState(ctx, "flow-key-old", entra.FlowState{State: "x", Nonce: "x", Verifier: "x", ExpiresAt: time.Now().Add(-time.Minute)})
	_ = store.PutState(ctx, "flow-key-2", fs)
	t.Cleanup(func() { _, _ = pool.Exec(ctx, `DELETE FROM sso_flow_state WHERE key LIKE 'flow-key-%'`) })
	if _, ok, _ := store.TakeState(ctx, "flow-key-old"); ok {
		t.Fatal("expired flow state survived the sweep")
	}
}

func TestSSOStoreSessions(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	store := NewSSOStore(pool)
	users := NewAPIUserRepo(pool)

	u, err := users.Insert(ctx, "sso-store-test@example.com", models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = users.Delete(ctx, u.ID) })
	uid := itoa(u.ID)

	now := time.Now().Round(time.Millisecond)
	s := entra.Session{UserID: uid, CreatedAt: now, LastSeenAt: now, ExpiresAt: now.Add(time.Hour)}
	if err := store.PutSession(ctx, "sess-1", s); err != nil {
		t.Fatal(err)
	}
	if err := store.PutSession(ctx, "sess-2", s); err != nil {
		t.Fatal(err)
	}

	got, ok, err := store.GetSession(ctx, "sess-1")
	if err != nil || !ok || got.UserID != uid || !got.ExpiresAt.Equal(s.ExpiresAt) {
		t.Fatalf("GetSession: %+v ok=%v err=%v", got, ok, err)
	}

	later := now.Add(10 * time.Minute)
	if err := store.TouchSession(ctx, "sess-1", later, later.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _, _ = store.GetSession(ctx, "sess-1")
	if !got.LastSeenAt.Equal(later) || !got.ExpiresAt.Equal(later.Add(time.Hour)) {
		t.Fatalf("TouchSession did not slide: %+v", got)
	}

	if err := store.DeleteSession(ctx, "sess-1"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.GetSession(ctx, "sess-1"); ok {
		t.Fatal("deleted session still present")
	}
	if err := store.DeleteUserSessions(ctx, uid); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := store.GetSession(ctx, "sess-2"); ok {
		t.Fatal("DeleteUserSessions left a session behind")
	}
	if _, ok, _ := store.GetSession(ctx, "never-existed"); ok {
		t.Fatal("unknown key reported ok")
	}
	if err := store.PutSession(ctx, "bad", entra.Session{UserID: "not-an-int"}); err == nil {
		t.Fatal("expected error for non-integer user id")
	}
}

func TestAPIUserRepoRolesAndEntra(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewAPIUserRepo(pool)
	mappings := NewSSORoleMappingRepo(pool)

	u, err := users.Insert(ctx, "Entra-Test@example.com", models.RoleEditor)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = users.Delete(ctx, u.ID) })
	if u.Role != models.RoleEditor || u.SSO {
		t.Fatalf("inserted user %+v", u)
	}

	if _, err := users.GetByEmailFold(ctx, "entra-test@EXAMPLE.com"); err != nil {
		t.Fatalf("GetByEmailFold: %v", err)
	}
	if err := users.LinkEntra(ctx, u.ID, "oid-test-1", "entra-test@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := users.SetRole(ctx, u.ID, models.RoleAdmin); err != nil {
		t.Fatal(err)
	}
	got, err := users.GetByEntraOID(ctx, "oid-test-1")
	if err != nil || got.ID != u.ID || !got.SSO || got.Role != models.RoleAdmin || got.EmailAddress != "entra-test@example.com" {
		t.Fatalf("GetByEntraOID: %+v err=%v", got, err)
	}
	if _, err := users.GetByEntraOID(ctx, "nope"); err != models.ErrAPIUserNotFound {
		t.Fatalf("unknown oid: %v", err)
	}
	auth, err := users.GetForAuthByID(ctx, u.ID)
	if err != nil || auth.Role != models.RoleAdmin || auth.EntraOID == nil || *auth.EntraOID != "oid-test-1" {
		t.Fatalf("GetForAuthByID: %+v err=%v", auth, err)
	}

	m, err := mappings.Upsert(ctx, "TB.Test", models.RoleViewer)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mappings.Delete(ctx, m.ID) })
	m2, err := mappings.Upsert(ctx, "TB.Test", models.RoleAdmin)
	if err != nil || m2.ID != m.ID || m2.Role != models.RoleAdmin {
		t.Fatalf("upsert did not update in place: %+v err=%v", m2, err)
	}
	list, err := mappings.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, x := range list {
		if x.ID == m.ID {
			found = true
		}
	}
	if !found {
		t.Fatal("mapping missing from List")
	}
	if err := mappings.Delete(ctx, m.ID); err != nil {
		t.Fatal(err)
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
