package server

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type Client struct {
	State       *appState
	Creds       *creds
	Config      *appConfig
	CWClient    *psa.Client
	WebexClient *webex.Client
	Pool        *pgxpool.Pool
	Queries     *db.Queries
	Server      *gin.Engine

	testing     bool
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

func Run(embeddedMigrations embed.FS) error {
	setInitialLogger()
	ctx := context.Background()

	cr := getCreds()
	if err := cr.validate(); err != nil {
		return fmt.Errorf("validating credentials: %w", err)
	}

	pool, err := setupDB(ctx, cr.PostgresDSN, embeddedMigrations)
	if err != nil {
		return fmt.Errorf("setting up db connections: %w", err)
	}

	cl := newClient(cr, pool, testingEnabled())

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

	cl.State, err = cl.getAppState(ctx)
	if err != nil {
		return fmt.Errorf("fetching app state: %w", err)
	}

	if err := cl.bootstrapAdmin(ctx); err != nil {
		return fmt.Errorf("bootstrapping initial admin key: %w", err)
	}

	if !cl.testing {
		if err := cl.initAllHooks(); err != nil {
			return fmt.Errorf("initializing webhooks: %w", err)
		}
	} else {
		slog.Debug("not initializing hooks since we are in mock mode")
	}

	return nil
}

func newClient(cr *creds, pool *pgxpool.Pool, testing bool) *Client {
	slog.Debug("initializing server client")

	cwCreds := &psa.Creds{
		PublicKey:  cr.CWPublicKey,
		PrivateKey: cr.CWPrivateKey,
		ClientId:   cr.CWClientID,
		CompanyId:  cr.CWCompanyID,
	}

	cl := &Client{
		State:       defaultAppState,
		Creds:       cr,
		Queries:     db.New(pool),
		Pool:        pool,
		CWClient:    psa.NewClient(cwCreds),
		WebexClient: webex.NewClient(cr.WebexSecret),
		testing:     testing,
	}

	return cl
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

func (c *creds) validate() error {
	req := map[string]string{
		"INITIAL_ADMIN_EMAIL": c.InitialAdminEmail,
		"POSTGRES_DSN":        c.PostgresDSN,
	}

	// values that are okay to be empty if in testing
	okTest := map[string]string{
		"ROOT_URL":      c.RootURL,
		"CW_PUB_KEY":    c.CWPublicKey,
		"CW_PRIV_KEY":   c.CWPrivateKey,
		"CW_CLIENT_ID":  c.CWClientID,
		"CW_COMPANY_ID": c.CWCompanyID,
		"WEBEX_SECRET":  c.WebexSecret,
	}

	var empty []string
	for k, v := range req {
		if v == "" {
			empty = append(empty, k)
		}
	}

	for k, v := range okTest {
		if v == "" {
			if testingEnabled() {
				slog.Warn("env variable empty, but ok since testing is enabled", "key", k)
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

func testingEnabled() bool {
	return os.Getenv("TESTING") == "true"
}
