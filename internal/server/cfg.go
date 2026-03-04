package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/thecoretg/ticketbot/internal/mock"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/repos"
	"github.com/thecoretg/ticketbot/internal/webex"
	"github.com/thecoretg/ticketbot/models"
)

type Creds struct {
	RootURL              string
	InitialAdminEmail    string
	InitialAdminPassword string
	PostgresDSN          string
	WebexAPISecret       string
	CWCreds              *psa.Creds
}

type TestFlags struct {
	APIKey          *string
	SkipAuth        bool
	SkipHooks       bool
	MockWebex       bool
	StoreTTLSeconds int64
}

func getCreds() *Creds {
	return &Creds{
		RootURL:              os.Getenv("ROOT_URL"),
		InitialAdminEmail:    os.Getenv("INITIAL_ADMIN_EMAIL"),
		InitialAdminPassword: os.Getenv("INITIAL_ADMIN_PASSWORD"),
		PostgresDSN:          os.Getenv("POSTGRES_DSN"),
		WebexAPISecret:       os.Getenv("WEBEX_SECRET"),
		CWCreds: &psa.Creds{
			PublicKey:  os.Getenv("CW_PUB_KEY"),
			PrivateKey: os.Getenv("CW_PRIV_KEY"),
			ClientId:   os.Getenv("CW_CLIENT_ID"),
			CompanyId:  os.Getenv("CW_COMPANY_ID"),
		},
	}
}

func (c *Creds) validate(tf *TestFlags) error {
	req := map[string]string{
		"INITIAL_ADMIN_EMAIL": c.InitialAdminEmail,
	}

	cwVals := map[string]string{
		"CW_PUB_KEY":    c.CWCreds.PublicKey,
		"CW_PRIV_KEY":   c.CWCreds.PrivateKey,
		"CW_CLIENT_ID":  c.CWCreds.ClientId,
		"CW_COMPANY_ID": c.CWCreds.CompanyId,
	}

	var empty []string
	for k, v := range req {
		if v == "" {
			empty = append(empty, k)
		}
	}

	if c.PostgresDSN == "" {
		empty = append(empty, "POSTGRES_DSN")
	}

	if c.RootURL == "" {
		if tf.SkipHooks {
			slog.Warn("ROOT_URL is empty, but ok since SKIP_HOOKS is enabled")
		} else {
			empty = append(empty, "ROOT_URL")
		}
	}

	if c.WebexAPISecret == "" {
		empty = append(empty, "WEBEX_SECRET")
	}

	for k, v := range cwVals {
		if v == "" {
			empty = append(empty, k)
		}
	}

	if len(empty) > 0 {
		return fmt.Errorf("1 or more required env variables are empty: %v", empty)
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

func makeMessageSender(mocking bool, webexSecret string) repos.MessageSender {
	if mocking {
		slog.Info("running with webex mocking")
		return mock.NewWebexClient(webexSecret)
	}

	return webex.NewClient(webexSecret)
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
		APIKey:          apiKey,
		SkipAuth:        os.Getenv("SKIP_AUTH") == "true",
		SkipHooks:       os.Getenv("SKIP_HOOKS") == "true",
		MockWebex:       os.Getenv("MOCK_WEBEX") == "true",
		StoreTTLSeconds: ttl,
	}
}
