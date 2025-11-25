package oldserver

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/external/psa"
	"github.com/thecoretg/ticketbot/internal/external/webex"
	"github.com/thecoretg/ticketbot/internal/mock"
	"github.com/thecoretg/ticketbot/migrations"
)

type Client struct {
	State         *AppState
	Creds         *creds
	Config        *AppConfig
	CWClient      *psa.Client
	MessageSender messageSender
	Pool          *pgxpool.Pool
	Queries       *db.Queries
	Server        *gin.Engine

	testing     testFlags
	ticketLocks sync.Map
}

type creds struct {
	RootURL           string
	InitialAdminEmail string
	PostgresDSN       string
	CWPublicKey       string
	CWPrivateKey      string
	CWClientID        string
	CWCompanyID       string
	WebexSecret       string
}

type testFlags struct {
	skipAuth        bool
	skipHooks       bool
	mockWebex       bool
	mockConnectwise bool // currently does nothing
}

func Run() error {
	setInitialLogger()
	ctx := context.Background()

	tf := getTestFlags()
	cr := getCreds()
	if err := cr.validate(tf); err != nil {
		return fmt.Errorf("validating credentials: %w", err)
	}

	embeddedMigrations, err := fs.Sub(migrations.Migrations, ".")
	if err != nil {
		return fmt.Errorf("accessing embedded migrations: %w", err)
	}

	pool, err := setupDB(ctx, cr.PostgresDSN, embeddedMigrations)
	if err != nil {
		return fmt.Errorf("setting up db connections: %w", err)
	}

	cl, err := newClient(cr, pool, tf)
	if err != nil {
		return fmt.Errorf("creating server client: %w", err)
	}

	if err := cl.startup(ctx); err != nil {
		return fmt.Errorf("running server startup: %w", err)
	}

	if err := cl.serve(); err != nil {
		return fmt.Errorf("serving api: %w", err)
	}

	return nil
}

func (cl *Client) serve() error {
	cl.Server = gin.Default()
	cl.addRoutes()

	slog.Debug("running server")
	return cl.Server.Run()
}

func (cl *Client) startup(ctx context.Context) error {
	var err error
	cl.Config, err = cl.getFullConfig(ctx)
	if err != nil {
		return fmt.Errorf("fetching config: %w", err)
	}
	setLogLevel(cl.Config.Debug)

	cl.setStateIfNil()
	if err := cl.bootstrapAdmin(ctx); err != nil {
		return fmt.Errorf("bootstrapping initial admin key: %w", err)
	}

	if !cl.testing.skipHooks {
		if err := cl.initAllHooks(); err != nil {
			return fmt.Errorf("initializing webhooks: %w", err)
		}
	} else {
		slog.Info("skipping webhook creating")
	}

	return nil
}

func newClient(cr *creds, pool *pgxpool.Pool, tf testFlags) (*Client, error) {
	slog.Debug("initializing server client")

	cwCreds := &psa.Creds{
		PublicKey:  cr.CWPublicKey,
		PrivateKey: cr.CWPrivateKey,
		ClientId:   cr.CWClientID,
		CompanyId:  cr.CWCompanyID,
	}

	ms, err := getMessageSender(cr.WebexSecret, tf.mockWebex)
	if err != nil {
		return nil, fmt.Errorf("getting message sender: %w", err)
	}

	cl := &Client{
		State:         defaultAppState,
		Creds:         cr,
		Queries:       db.New(pool),
		Pool:          pool,
		CWClient:      psa.NewClient(cwCreds),
		MessageSender: ms,
		testing:       tf,
	}

	return cl, nil
}

func getMessageSender(token string, mocking bool) (messageSender, error) {
	if mocking {
		slog.Info("using mock webex client")
		return mock.NewWebexClient(token), nil
	}

	if token == "" {
		return nil, errors.New("webex secret is empty and testing is not enabled")
	}

	return webex.NewClient(token), nil
}

func getCreds() *creds {
	return &creds{
		RootURL:           os.Getenv("ROOT_URL"),
		InitialAdminEmail: os.Getenv("INITIAL_ADMIN_EMAIL"),
		PostgresDSN:       os.Getenv("POSTGRES_DSN"),
		CWPublicKey:       os.Getenv("CW_PUB_KEY"),
		CWPrivateKey:      os.Getenv("CW_PRIV_KEY"),
		CWClientID:        os.Getenv("CW_CLIENT_ID"),
		CWCompanyID:       os.Getenv("CW_COMPANY_ID"),
		WebexSecret:       os.Getenv("WEBEX_SECRET"),
	}
}

func (c *creds) validate(tf testFlags) error {
	req := map[string]string{
		"INITIAL_ADMIN_EMAIL": c.InitialAdminEmail,
		"POSTGRES_DSN":        c.PostgresDSN,
	}

	cwVals := map[string]string{
		"CW_PUB_KEY":    c.CWPublicKey,
		"CW_PRIV_KEY":   c.CWPrivateKey,
		"CW_CLIENT_ID":  c.CWClientID,
		"CW_COMPANY_ID": c.CWCompanyID,
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

func getTestFlags() testFlags {
	return testFlags{
		skipAuth:        os.Getenv("SKIP_AUTH") == "true",
		skipHooks:       os.Getenv("SKIP_HOOKS") == "true",
		mockWebex:       os.Getenv("MOCK_WEBEX") == "true",
		mockConnectwise: os.Getenv("MOCK_CONNECTWISE") == "true",
	}
}
