package newserver

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/internal/service/config"
	"github.com/thecoretg/ticketbot/internal/service/cwsvc"
	"github.com/thecoretg/ticketbot/internal/service/notifier"
	"github.com/thecoretg/ticketbot/internal/service/ticketbot"
	"github.com/thecoretg/ticketbot/internal/service/user"
	"github.com/thecoretg/ticketbot/internal/service/webexsvc"
)

type App struct {
	Creds       *creds
	TestFlags   *testFlags
	Stores      *models.AllRepos
	CWClient    *psa.Client
	WebexClient *webex.Client
	Pool        *pgxpool.Pool
	Config      *models.Config
	Svc         *Services
}

type creds struct {
	RootURL           string
	InitialAdminEmail string
	PostgresDSN       string
	WebexSecret       string
	cw                *psa.Creds
}

type testFlags struct {
	skipAuth        bool
	skipHooks       bool
	mockWebex       bool
	mockConnectwise bool // currently does nothing
}

type Services struct {
	Config    *config.Service
	User      *user.Service
	CW        *cwsvc.Service
	Webex     *webexsvc.Service
	Notifier  *notifier.Service
	Ticketbot *ticketbot.Service
}

type repoType string

const (
	RepoTypePostgres repoType = "POSTGRES"
)

func Run() error {
	ctx := context.Background()
	a, err := NewApp(ctx)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	srv := gin.Default()
	a.addRoutes(srv)

	return srv.Run()
}

func NewApp(ctx context.Context) (*App, error) {
	cr := getCreds()
	tf := getTestFlags()
	if err := cr.validate(tf); err != nil {
		return nil, fmt.Errorf("validating credentials: %w", err)
	}

	cw := psa.NewClient(cr.cw)
	wx := webex.NewClient(cr.WebexSecret)

	s, err := initStores(ctx, cr)
	if err != nil {
		return nil, fmt.Errorf("initializing stores: %w", err)
	}
	r := s.stores

	cs := config.New(r.Config)
	cfg, err := cs.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("getting initial config: %w", err)
	}

	nr := notifier.Repos{
		Rooms:         r.WebexRoom,
		Notifiers:     r.Notifiers,
		Notifications: r.Notifications,
		Forwards:      r.Forwards,
	}

	us := user.New(r.APIUser, r.APIKey)
	cws := cwsvc.New(s.pool, r.CW, cw)
	ws := webexsvc.New(s.pool, r.WebexRoom, wx)

	ns := notifier.New(*cfg, nr, wx, cr.cw.CompanyId, cfg.MaxMessageLength)
	tb := ticketbot.New(*cfg, cws, ns)
	return &App{
		Creds:       cr,
		TestFlags:   tf,
		Stores:      r,
		Pool:        s.pool,
		CWClient:    psa.NewClient(cr.cw),
		WebexClient: webex.NewClient(cr.WebexSecret),
		Svc: &Services{
			Config:    cs,
			User:      us,
			CW:        cws,
			Webex:     ws,
			Notifier:  ns,
			Ticketbot: tb,
		},
	}, nil
}

func getCreds() *creds {
	return &creds{
		RootURL:           os.Getenv("ROOT_URL"),
		InitialAdminEmail: os.Getenv("INITIAL_ADMIN_EMAIL"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		WebexSecret:       os.Getenv("WEBEX_SECRET"),
		cw: &psa.Creds{
			PublicKey:  os.Getenv("CW_PUB_KEY"),
			PrivateKey: os.Getenv("CW_PRIV_KEY"),
			ClientId:   os.Getenv("CW_CLIENT_ID"),
			CompanyId:  os.Getenv("CW_COMPANY_ID"),
		},
	}
}

func (c *creds) validate(tf *testFlags) error {
	req := map[string]string{
		"INITIAL_ADMIN_EMAIL": c.InitialAdminEmail,
		"POSTGRES_DSN":        c.PostgresDSN,
	}

	cwVals := map[string]string{
		"CW_PUB_KEY":    c.cw.PublicKey,
		"CW_PRIV_KEY":   c.cw.PrivateKey,
		"CW_CLIENT_ID":  c.cw.ClientId,
		"CW_COMPANY_ID": c.cw.CompanyId,
	}

	var empty []string
	for k, v := range req {
		if v == "" {
			empty = append(empty, k)
		}
	}

	if c.RootURL == "" {
		if tf.skipHooks {
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
			if tf.mockConnectwise {
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

func getTestFlags() *testFlags {
	return &testFlags{
		skipAuth:        os.Getenv("SKIP_AUTH") == "true",
		skipHooks:       os.Getenv("SKIP_HOOKS") == "true",
		mockWebex:       os.Getenv("MOCK_WEBEX") == "true",
		mockConnectwise: os.Getenv("MOCK_CONNECTWISE") == "true",
	}
}
