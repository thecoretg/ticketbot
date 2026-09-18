package sso

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

// fakeUsers satisfies repos.APIUserRepository via embedding; only the methods Provision and
// Lookup call are implemented.
type fakeUsers struct {
	repos.APIUserRepository
	byID   map[int]*models.APIUser
	oids   map[int]string
	nextID int
}

func newFakeUsers() *fakeUsers {
	return &fakeUsers{byID: map[int]*models.APIUser{}, oids: map[int]string{}, nextID: 1}
}

func (f *fakeUsers) Get(_ context.Context, id int) (*models.APIUser, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, models.ErrAPIUserNotFound
}

func (f *fakeUsers) GetByEntraOID(_ context.Context, oid string) (*models.APIUser, error) {
	for id, o := range f.oids {
		if o == oid {
			return f.byID[id], nil
		}
	}
	return nil, models.ErrAPIUserNotFound
}

func (f *fakeUsers) GetByEmailFold(_ context.Context, email string) (*models.APIUser, error) {
	for _, u := range f.byID {
		if strings.EqualFold(u.EmailAddress, email) {
			return u, nil
		}
	}
	return nil, models.ErrAPIUserNotFound
}

func (f *fakeUsers) Insert(_ context.Context, email string, role models.Role) (*models.APIUser, error) {
	u := &models.APIUser{ID: f.nextID, EmailAddress: email, Role: role}
	f.byID[u.ID] = u
	f.nextID++
	return u, nil
}

func (f *fakeUsers) SetRole(_ context.Context, id int, role models.Role) error {
	f.byID[id].Role = role
	return nil
}

func (f *fakeUsers) LinkEntra(_ context.Context, id int, oid, email string) error {
	f.oids[id] = oid
	f.byID[id].EmailAddress = email
	f.byID[id].SSO = true
	return nil
}

type fakeMappings struct {
	repos.SSORoleMappingRepository
	list []*models.SSORoleMapping
}

func (f *fakeMappings) List(context.Context) ([]*models.SSORoleMapping, error) { return f.list, nil }

func newService(t *testing.T, users *fakeUsers, mappings ...*models.SSORoleMapping) *Service {
	t.Helper()
	s, err := New(context.Background(), Params{
		Users:    users,
		Mappings: &fakeMappings{list: mappings},
		Cfg:      &models.Config{},
		RootURL:  "https://tb.example.com/",
	})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mapping(entraRole string, role models.Role) *models.SSORoleMapping {
	return &models.SSORoleMapping{EntraRole: entraRole, Role: role}
}

func TestAuthorize(t *testing.T) {
	s := newService(t, newFakeUsers(), mapping("TB.Admin", models.RoleAdmin), mapping("TB.Viewer", models.RoleViewer))

	if err := s.Authorize(entra.Claims{Roles: []string{"TB.Viewer"}}); err != nil {
		t.Errorf("mapped role refused: %v", err)
	}
	err := s.Authorize(entra.Claims{Roles: []string{"Unrelated"}})
	if !errors.Is(err, entra.ErrNotAuthorized) {
		t.Errorf("unmapped role allowed: %v", err)
	}
	if err := s.Authorize(entra.Claims{}); !errors.Is(err, entra.ErrNotAuthorized) {
		t.Errorf("no roles allowed: %v", err)
	}
}

func TestProvisionCreatesWithHighestRole(t *testing.T) {
	users := newFakeUsers()
	s := newService(t, users, mapping("TB.Admin", models.RoleAdmin), mapping("TB.Viewer", models.RoleViewer))

	id, err := s.Provision(context.Background(), entra.Claims{OID: "oid-1", PreferredUsername: "Ann@Example.com", Roles: []string{"TB.Viewer", "TB.Admin"}})
	if err != nil {
		t.Fatal(err)
	}
	if id != "1" {
		t.Fatalf("id = %q, want 1", id)
	}
	u := users.byID[1]
	if u.Role != models.RoleAdmin || !u.SSO || u.EmailAddress != "Ann@Example.com" {
		t.Fatalf("unexpected user %+v", u)
	}
}

func TestProvisionLinksLocalAccountByEmailOnce(t *testing.T) {
	users := newFakeUsers()
	local, _ := users.Insert(context.Background(), "admin@example.com", models.RoleAdmin)
	s := newService(t, users, mapping("TB.Viewer", models.RoleViewer))

	id, err := s.Provision(context.Background(), entra.Claims{OID: "oid-a", PreferredUsername: "Admin@example.com", Roles: []string{"TB.Viewer"}})
	if err != nil {
		t.Fatal(err)
	}
	if id != "1" || users.oids[local.ID] != "oid-a" {
		t.Fatalf("local account not linked: id=%s oids=%v", id, users.oids)
	}
	if local.Role != models.RoleViewer {
		t.Fatalf("role not taken from entra: %s", local.Role)
	}

	// A second identity with the same email must not take over the linked account.
	_, err = s.Provision(context.Background(), entra.Claims{OID: "oid-b", PreferredUsername: "admin@example.com", Roles: []string{"TB.Viewer"}})
	if !errors.Is(err, ErrEmailLinked) {
		t.Fatalf("expected ErrEmailLinked, got %v", err)
	}
}

func TestProvisionRefreshesRoleOnReturn(t *testing.T) {
	users := newFakeUsers()
	s := newService(t, users, mapping("TB.Admin", models.RoleAdmin), mapping("TB.Editor", models.RoleEditor))

	if _, err := s.Provision(context.Background(), entra.Claims{OID: "oid-1", Email: "b@example.com", Roles: []string{"TB.Admin"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Provision(context.Background(), entra.Claims{OID: "oid-1", Email: "b@example.com", Roles: []string{"TB.Editor"}}); err != nil {
		t.Fatal(err)
	}
	if users.byID[1].Role != models.RoleEditor {
		t.Fatalf("role = %s, want editor", users.byID[1].Role)
	}
	if len(users.byID) != 1 {
		t.Fatalf("expected one user, got %d", len(users.byID))
	}
}

func TestProvisionWithoutMappedRole(t *testing.T) {
	s := newService(t, newFakeUsers())
	_, err := s.Provision(context.Background(), entra.Claims{OID: "x", Email: "x@example.com", Roles: []string{"Nope"}})
	if !errors.Is(err, entra.ErrNotAuthorized) {
		t.Fatalf("expected not authorized, got %v", err)
	}
}

func TestLookup(t *testing.T) {
	users := newFakeUsers()
	users.Insert(context.Background(), "a@example.com", models.RoleViewer)
	s := newService(t, users)

	if u, err := s.Lookup(context.Background(), "1"); err != nil || u.ID != 1 {
		t.Fatalf("Lookup(1) = %v, %v", u, err)
	}
	if _, err := s.Lookup(context.Background(), "2"); !errors.Is(err, entra.ErrUserNotFound) {
		t.Fatalf("missing user: %v", err)
	}
	if _, err := s.Lookup(context.Background(), "abc"); !errors.Is(err, entra.ErrUserNotFound) {
		t.Fatalf("garbage id: %v", err)
	}
}

func TestRedirectURIWithoutAuth(t *testing.T) {
	s := newService(t, newFakeUsers())
	if got := s.RedirectURI(); got != "https://tb.example.com/auth/sso/callback" {
		t.Fatalf("RedirectURI = %q", got)
	}
	if err := s.TestConnection(context.Background()); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("TestConnection without auth: %v", err)
	}
}
