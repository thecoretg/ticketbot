package env

import (
	"strings"
	"testing"
	"time"
)

func setRequired(t *testing.T) {
	t.Helper()
	for k, v := range map[string]string{
		"POSTGRES_DSN":        "dsn",
		"INITIAL_ADMIN_EMAIL": "a@b.c",
		"WEBEX_SECRET":        "wx",
		"CW_PUB_KEY":          "pub",
		"CW_PRIV_KEY":         "priv",
		"CW_CLIENT_ID":        "cid",
		"CW_COMPANY_ID":       "co",
		"SKIP_HOOKS":          "true",
	} {
		t.Setenv(k, v)
	}
}

func TestLoadDefaults(t *testing.T) {
	setRequired(t)
	e, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if e.Port != "8080" || e.StoreTTL != 15*time.Minute || e.PostgresMaxConns != 10 {
		t.Fatalf("unexpected defaults: %+v", e)
	}
}

func TestLoadReportsAllProblems(t *testing.T) {
	setRequired(t)
	t.Setenv("SKIP_HOOKS", "")
	t.Setenv("WEBEX_SECRET", "")
	t.Setenv("STORE_TTL_SECONDS", "abc")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"ROOT_URL", "WEBEX_SECRET", "STORE_TTL_SECONDS"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %s", err, want)
		}
	}
}

func TestLoadRejectsNonPositiveTTL(t *testing.T) {
	setRequired(t)
	t.Setenv("STORE_TTL_SECONDS", "0")
	if _, err := Load(); err == nil {
		t.Fatal("expected error for zero TTL")
	}
}

func TestLoadEntraAllOrNothing(t *testing.T) {
	setRequired(t)
	t.Setenv("ENTRA_TENANT_ID", "11111111-1111-1111-1111-111111111111")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error when only one ENTRA_* variable is set")
	}
	for _, want := range []string{"ENTRA_CLIENT_ID", "ENTRA_CLIENT_SECRET", "ROOT_URL"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %s", err, want)
		}
	}

	t.Setenv("ENTRA_CLIENT_ID", "cid")
	t.Setenv("ENTRA_CLIENT_SECRET", "sec")
	t.Setenv("ROOT_URL", "http://localhost:8080")
	e, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if !e.Entra.Configured() {
		t.Fatal("expected Entra to be configured")
	}
}

func TestLoadWithoutEntra(t *testing.T) {
	setRequired(t)
	e, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if e.Entra.Configured() {
		t.Fatal("expected Entra to be unconfigured")
	}
}

func TestLoadNormalizesRootURL(t *testing.T) {
	setRequired(t)
	cases := map[string]string{
		"ticketbot.example.com":          "https://ticketbot.example.com",
		"https://ticketbot.example.com/": "https://ticketbot.example.com",
		"http://localhost:8080":          "http://localhost:8080",
	}
	for in, want := range cases {
		t.Setenv("ROOT_URL", in)
		e, err := Load()
		if err != nil {
			t.Fatal(err)
		}
		if e.RootURL != want {
			t.Errorf("ROOT_URL=%q -> %q, want %q", in, e.RootURL, want)
		}
	}
}
