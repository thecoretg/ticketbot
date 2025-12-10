package server

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/mock"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/syncsvc"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
	"github.com/thecoretg/ticketbot/internal/service/webhooks"
	"github.com/thecoretg/ticketbot/pkg/psa"
	"github.com/thecoretg/ticketbot/pkg/webex"
)

type App struct {
	Creds                   *Creds
	TestFlags               *TestFlags
	Stores                  *models.AllRepos
	CWClient                *psa.Client
	MessageSender           models.MessageSender
	Pool                    *pgxpool.Pool
	Config                  *models.Config
	Svc                     *Services
	CurrentMigrationVersion int64
}

type Creds struct {
	RootURL           string
	InitialAdminEmail string
	PostgresDSN       string
	WebexSecret       string
	CWCreds           *psa.Creds
}

type TestFlags struct {
	APIKey          *string
	SkipAuth        bool
	SkipHooks       bool
	MockWebex       bool
	MockConnectwise bool // currently does nothing
}

type Services struct {
	Config    *config.Service
	User      *user.Service
	CW        *cwsvc.Service
	Hooks     *webhooks.Service
	Webex     *webexsvc.Service
	Sync      *syncsvc.Service
	Notifier  *notifier.Service
	Ticketbot *ticketbot.Service
}

func NewApp(ctx context.Context, migVersion int64) (*App, error) {
	cr := getCreds()
	tf := getTestFlags()
	if err := cr.validate(tf); err != nil {
		return nil, fmt.Errorf("validating credentials: %w", err)
	}

	cw := psa.NewClient(cr.CWCreds)
	ms := makeMessageSender(tf.MockWebex, cr.WebexSecret)

	s, err := CreateStores(ctx, cr, migVersion)
	if err != nil {
		return nil, fmt.Errorf("initializing stores: %w", err)
	}
	r := s.Repos

	cfg, _ := r.Config.Get(ctx)
	if cfg == nil {
		cfg = &models.DefaultConfig
	}

	cfg = loadConfigOverrides(cfg)

	cs := config.New(r.Config, cfg)

	us := user.New(r.APIUser, r.APIKey)
	cws := cwsvc.New(s.Pool, r.CW, cw)
	ws := webexsvc.New(s.Pool, r.WebexRecipients, ms)
	wh := webhooks.New(cw, cr.RootURL)

	nr := notifier.SvcParams{
		Cfg:              cfg,
		WebexSvc:         ws,
		NotifierRules:    r.NotifierRules,
		Notifications:    r.TicketNotifications,
		Forwards:         r.NotifierForwards,
		Pool:             s.Pool,
		MessageSender:    ms,
		CWCompanyID:      cr.CWCreds.CompanyId,
		MaxMessageLength: cfg.MaxMessageLength,
	}

	ns := notifier.New(nr)
	sns := syncsvc.New(s.Pool, cws, ws, ns)

	tb := ticketbot.New(cfg, cws, ns)
	return &App{
		Creds:         cr,
		Config:        cfg,
		TestFlags:     tf,
		Stores:        r,
		Pool:          s.Pool,
		CWClient:      psa.NewClient(cr.CWCreds),
		MessageSender: webex.NewClient(cr.WebexSecret),
		Svc: &Services{
			Config:    cs,
			User:      us,
			Hooks:     wh,
			CW:        cws,
			Webex:     ws,
			Sync:      sns,
			Notifier:  ns,
			Ticketbot: tb,
		},
	}, nil
}

func getCreds() *Creds {
	return &Creds{
		RootURL:           os.Getenv("ROOT_URL"),
		InitialAdminEmail: os.Getenv("INITIAL_ADMIN_EMAIL"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		WebexSecret:       os.Getenv("WEBEX_SECRET"),
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

	if c.WebexSecret == "" {
		empty = append(empty, "WEBEX_SECRET")
	}

	for k, v := range cwVals {
		if v == "" {
			if tf.MockConnectwise {
				slog.Warn("env variable empty, but ok since MOCK_CONNECTWISE is enabled", "key", k)
				continue
			}
			empty = append(empty, k)
		}
	}

	if len(empty) > 0 {
		return fmt.Errorf("1 or more required env variables are empty: %v", empty)
	}

	return nil
}

func makeMessageSender(mocking bool, webexSecret string) models.MessageSender {
	if mocking {
		slog.Info("running with webex mocking")
		return mock.NewWebexClient(webexSecret)
	}

	return webex.NewClient(webexSecret)
}

func getTestFlags() *TestFlags {
	var apiKey *string
	if os.Getenv("API_KEY") != "" {
		k := os.Getenv("API_KEY")
		apiKey = &k
	}

	return &TestFlags{
		APIKey:          apiKey,
		SkipAuth:        os.Getenv("SKIP_AUTH") == "true",
		SkipHooks:       os.Getenv("SKIP_HOOKS") == "true",
		MockWebex:       os.Getenv("MOCK_WEBEX") == "true",
		MockConnectwise: os.Getenv("MOCK_CONNECTWISE") == "true",
	}
}
