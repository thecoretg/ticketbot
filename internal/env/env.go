// Package env is the single place the process reads its environment. Load parses every variable
// the app uses, applies defaults, and reports all missing or malformed values at once.
package env

import (
	"errors"
	"fmt"
	"os"
	"strconv"
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

	// Local testing
	SkipHooks bool
	MockWebex bool
	StoreTTL  time.Duration
}

// Load reads and validates the environment. It returns one error listing every problem.
func Load() (*Env, error) {
	var errs []error

	e := &Env{
		Port:                 stringOr("PORT", defaultPort),
		Debug:                boolVar("DEBUG"),
		PostgresDSN:          os.Getenv("POSTGRES_DSN"),
		RootURL:              os.Getenv("ROOT_URL"),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
		WebexSecret:          os.Getenv("WEBEX_SECRET"),
		CW: psa.Config{
			PublicKey:  os.Getenv("CW_PUB_KEY"),
			PrivateKey: os.Getenv("CW_PRIV_KEY"),
			ClientID:   os.Getenv("CW_CLIENT_ID"),
			CompanyID:  os.Getenv("CW_COMPANY_ID"),
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
