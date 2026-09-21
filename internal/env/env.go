// Package env is the single place the process reads its environment. Load parses every variable
// the app uses, applies defaults, and reports all missing or malformed values at once.
package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thecoretg/tctg-go/connectwise/psa"
)

const (
	defaultPort     = "8080"
	defaultStoreTTL = 15 * time.Minute
	defaultMaxConns = int32(10)
)

// Env is the parsed process environment. See .env.example for the variable list.
type Env struct {
	// Server
	Port  string
	Debug bool

	// Database
	PostgresDSN      string
	PostgresMaxConns int32

	// App
	RootURL              string
	InitialAdminEmail    string
	InitialAdminPassword string

	// Integrations
	CW          psa.Config
	WebexSecret string
	Entra       Entra

	// Local testing
	SkipHooks bool
	MockWebex bool
	StoreTTL  time.Duration

	// HookSignatureEnforced rejects ConnectWise callbacks whose signature does not verify. Only
	// HOOK_SIGNATURE_MODE=log turns it off, for diagnosing a signing problem during the parallel run.
	HookSignatureEnforced bool
}

// Entra is the Microsoft Entra ID app registration used for single sign-on. Set all three
// variables or none: with none the app runs without SSO and the dashboard's SSO page says so.
type Entra struct {
	TenantID     string
	ClientID     string
	ClientSecret string
}

// Configured reports whether SSO credentials were supplied. After Load it is all-or-nothing.
func (e Entra) Configured() bool {
	return e.TenantID != "" && e.ClientID != "" && e.ClientSecret != ""
}

// Load reads and validates the environment. It returns one error listing every problem.
func Load() (*Env, error) {
	var errs []error

	e := &Env{
		Port:                 stringOr("PORT", defaultPort),
		Debug:                boolVar("DEBUG"),
		PostgresDSN:          os.Getenv("POSTGRES_DSN"),
		RootURL:              normalizeRootURL(os.Getenv("ROOT_URL")),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
		WebexSecret:          os.Getenv("WEBEX_SECRET"),
		CW: psa.Config{
			PublicKey:  os.Getenv("CW_PUB_KEY"),
			PrivateKey: os.Getenv("CW_PRIV_KEY"),
			ClientID:   os.Getenv("CW_CLIENT_ID"),
			CompanyID:  os.Getenv("CW_COMPANY_ID"),
		},
		Entra: Entra{
			TenantID:     os.Getenv("ENTRA_TENANT_ID"),
			ClientID:     os.Getenv("ENTRA_CLIENT_ID"),
			ClientSecret: os.Getenv("ENTRA_CLIENT_SECRET"),
		},
		SkipHooks: boolVar("SKIP_HOOKS"),
		MockWebex: boolVar("MOCK_WEBEX"),
	}

	maxConns, err := intOr("POSTGRES_MAX_CONNS", int(defaultMaxConns))
	if err != nil {
		errs = append(errs, err)
	} else if maxConns <= 0 {
		errs = append(errs, errors.New("POSTGRES_MAX_CONNS must be greater than 0"))
	}
	e.PostgresMaxConns = int32(maxConns)

	ttlSecs, err := intOr("STORE_TTL_SECONDS", int(defaultStoreTTL.Seconds()))
	if err != nil {
		errs = append(errs, err)
	} else if ttlSecs <= 0 {
		errs = append(errs, errors.New("STORE_TTL_SECONDS must be greater than 0"))
	}
	e.StoreTTL = time.Duration(ttlSecs) * time.Second

	switch mode := strings.ToLower(strings.TrimSpace(os.Getenv("HOOK_SIGNATURE_MODE"))); mode {
	case "", "enforce":
		e.HookSignatureEnforced = true
	case "log":
		e.HookSignatureEnforced = false
	default:
		errs = append(errs, fmt.Errorf("HOOK_SIGNATURE_MODE must be enforce or log (got %q)", mode))
	}

	required := []struct{ name, val string }{
		{"POSTGRES_DSN", e.PostgresDSN},
		{"INITIAL_ADMIN_EMAIL", e.InitialAdminEmail},
		{"WEBEX_SECRET", e.WebexSecret},
		{"CW_PUB_KEY", e.CW.PublicKey},
		{"CW_PRIV_KEY", e.CW.PrivateKey},
		{"CW_CLIENT_ID", e.CW.ClientID},
		{"CW_COMPANY_ID", e.CW.CompanyID},
	}
	if !e.SkipHooks {
		required = append(required, struct{ name, val string }{"ROOT_URL", e.RootURL})
	}
	// SSO is all-or-nothing, and the redirect URI is built from ROOT_URL.
	if e.Entra.TenantID != "" || e.Entra.ClientID != "" || e.Entra.ClientSecret != "" {
		required = append(required,
			struct{ name, val string }{"ENTRA_TENANT_ID", e.Entra.TenantID},
			struct{ name, val string }{"ENTRA_CLIENT_ID", e.Entra.ClientID},
			struct{ name, val string }{"ENTRA_CLIENT_SECRET", e.Entra.ClientSecret},
			struct{ name, val string }{"ROOT_URL", e.RootURL},
		)
	}

	var missing []string
	for _, r := range required {
		if r.val == "" {
			missing = append(missing, r.name)
		}
	}
	if len(missing) > 0 {
		errs = append(errs, fmt.Errorf("required env variables are empty: %v", missing))
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return e, nil
}

// normalizeRootURL turns ROOT_URL into an origin with a scheme and no trailing slash. A bare
// host gets https, so both "ticketbot.example.com" and "https://ticketbot.example.com/" work.
func normalizeRootURL(v string) string {
	v = strings.TrimRight(strings.TrimSpace(v), "/")
	if v != "" && !strings.Contains(v, "://") {
		v = "https://" + v
	}
	return v
}

func stringOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func boolVar(key string) bool {
	return os.Getenv(key) == "true"
}

// intOr returns def when key is unset. A set but non-integer value is an error rather than a
// silent fallback so a typo cannot go unnoticed.
func intOr(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be an integer, got %q", key, v)
	}
	return i, nil
}
