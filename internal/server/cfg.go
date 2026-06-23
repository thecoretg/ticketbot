package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/thecoretg/tctg-go/connectwise/psa"
	"github.com/thecoretg/tctg-go/webex"
	"github.com/thecoretg/ticketbot/internal/mock"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/models"
)

type Creds struct {
	RootURL              string
	InitialAdminEmail    string
	InitialAdminPassword string
	PostgresDSN          string
	WebexAPISecret       string
	CWCreds              *psa.Config
}

type TestFlags struct {
	APIKey                   *string
	SkipAuth                 bool
	SkipHooks                bool
	MockWebex                bool
	SkipInitialPasswordReset bool
	StoreTTLSeconds          int64
}

func getCreds() *Creds {
	return &Creds{
		RootURL:              os.Getenv("ROOT_URL"),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
		PostgresDSN:          os.Getenv("POSTGRES_DSN"),
		WebexAPISecret:       os.Getenv("WEBEX_SECRET"),
		CWCreds: &psa.Config{
			PublicKey:  os.Getenv("CW_PUB_KEY"),
			PrivateKey: os.Getenv("CW_PRIV_KEY"),
			ClientID:   os.Getenv("CW_CLIENT_ID"),
			CompanyID:  os.Getenv("CW_COMPANY_ID"),
		},
	}
}

// validateBootstrap checks the env variables that must be present before the app
// can reach its database and seed the first admin. These genuinely cannot live in
// the app_config table (chicken-and-egg), so they stay env-only.
func (c *Creds) validateBootstrap() error {
	var empty []string
	if c.InitialAdminEmail == "" {
		empty = append(empty, "INITIAL_ADMIN_EMAIL")
	}
	if c.PostgresDSN == "" {
		empty = append(empty, "POSTGRES_DSN")
	}
	if len(empty) > 0 {
		return fmt.Errorf("1 or more required env variables are empty: %v", empty)
	}
	return nil
}

// mergeEnvConfig overlays env vars onto the stored config: when an env var is set
// it wins and is written back to the config so the database keeps a working copy
// (after which the env var can be dropped). Covers the credentials plus a few
// operational settings (bot identifier, killswitches, debug logging) that are
// handy to pin at deploy time. Returns the JSON keys of the fields that were
// sourced from the environment, so the admin panel can lock them.
func mergeEnvConfig(cfg *models.Config, c *Creds) []string {
	var locked []string

	setStr := func(key, val string, dst *string) {
		if val != "" {
			*dst = val
			locked = append(locked, key)
		}
	}
	setBool := func(key, envName string, dst *bool) {
		v := os.Getenv(envName)
		if v == "" {
			return
		}
		b, err := strconv.ParseBool(v)
		if err != nil {
			slog.Warn("ignoring non-boolean env var", "var", envName, "value", v)
			return
		}
		*dst = b
		locked = append(locked, key)
	}

	// Credentials.
	setStr("root_url", c.RootURL, &cfg.RootURL)
	setStr("cw_company_id", c.CWCreds.CompanyID, &cfg.CwCompanyID)
	setStr("cw_client_id", c.CWCreds.ClientID, &cfg.CwClientID)
	setStr("cw_public_key", c.CWCreds.PublicKey, &cfg.CwPublicKey)
	setStr("cw_private_key", c.CWCreds.PrivateKey, &cfg.CwPrivateKey)
	setStr("webex_secret", c.WebexAPISecret, &cfg.WebexSecret)

	// Operational settings.
	setStr("cw_bot_member_identifier", os.Getenv("CW_BOT_MEMBER_IDENTIFIER"), &cfg.CwBotMemberIdentifier)
	setBool("attempt_notify", "ATTEMPT_NOTIFY", &cfg.AttemptNotify)
	setBool("attempt_workflow", "ATTEMPT_WORKFLOW", &cfg.AttemptWorkflow)
	setBool("debug_logging", "DEBUG_LOGGING", &cfg.DebugLogging)

	return locked
}

// validateCreds checks the effective (env-or-db) credentials needed to talk to
// Connectwise and Webex are present, so startup fails fast with a clear message
// rather than at first API call.
func validateCreds(cfg *models.Config, tf *TestFlags) error {
	vals := map[string]string{
		"cw_company_id":  cfg.CwCompanyID,
		"cw_client_id":   cfg.CwClientID,
		"cw_public_key":  cfg.CwPublicKey,
		"cw_private_key": cfg.CwPrivateKey,
		"webex_secret":   cfg.WebexSecret,
	}

	var empty []string
	for k, v := range vals {
		if v == "" {
			empty = append(empty, k)
		}
	}

	if cfg.RootURL == "" {
		if tf.SkipHooks {
			slog.Warn("root_url is empty, but ok since SKIP_HOOKS is enabled")
		} else {
			empty = append(empty, "root_url")
		}
	}

	if len(empty) > 0 {
		return fmt.Errorf("missing credentials (set via env or the Config menu): %v", empty)
	}

	return nil
}

// getStartupConfig gets the current config at server startup. It uses the default if one is not
// found in the store, upserts, and then returns the final result.
func getStartupConfig(ctx context.Context, r repos.ConfigRepository) (*models.Config, error) {
	// get initial config; if none in db, use default
	cfg, _ := r.Get(ctx)
	if cfg == nil {
		slog.Info("no config found in store; using default")
		cfg = &models.DefaultConfig
	}

	cfg, err := r.Upsert(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("upserting config: %w", err)
	}

	return cfg, nil
}

func makeMessageSender(ctx context.Context, mocking bool, webexSecret string) (repos.MessageSender, error) {
	if mocking {
		slog.Info("running with webex mocking")
		return mock.NewWebexClient(ctx, webexSecret)
	}

	return webex.NewClient(ctx, webex.Config{Token: webexSecret})
}

func getTestFlags() *TestFlags {
	var ttl int64
	var apiKey *string
	if os.Getenv("API_KEY") != "" {
		k := os.Getenv("API_KEY")
		apiKey = &k
	}

	ttlStr := os.Getenv("STORE_TTL_SECONDS")
	if ttlStr != "" {
		i, err := strconv.Atoi(ttlStr)
		if err != nil {
			slog.Error("couldn't convert STORE_TTL_SECONDS env var to integer, using default", "string", ttlStr)
		} else {
			ttl = int64(i)
			slog.Info("ttl test flag provided", "ttl", ttl)
		}
	}

	return &TestFlags{
		APIKey:                   apiKey,
		SkipAuth:                 os.Getenv("SKIP_AUTH") == "true",
		SkipHooks:                os.Getenv("SKIP_HOOKS") == "true",
		MockWebex:                os.Getenv("MOCK_WEBEX") == "true",
		SkipInitialPasswordReset: os.Getenv("SKIP_INITIAL_PASSWORD_RESET") == "true",
		StoreTTLSeconds:          ttl,
	}
}
