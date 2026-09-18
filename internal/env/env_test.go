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
