package server

import (
	"context"
	"embed"
	"errors"
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

var mocking = false

type Client struct {
	State       *appState
	Creds       *creds
	Config      *appConfig
	CWClient    *psa.Client
	WebexClient *webex.Client
	Pool        *pgxpool.Pool
	Queries     *db.Queries
	Server      *gin.Engine

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

	if os.Getenv("MOCKING") == "true" {
		slog.Info("server started in mock mode")
		mocking = true
	}

	root := os.Getenv("ROOT_URL")
	if root == "" {
		return errors.New("root URL is empty")
	}

	cr := getCreds()
	if err := cr.validate(); err != nil {
		return fmt.Errorf("validating credentials: %w", err)
	}

	pool, err := setupDB(ctx, cr.PostgresDSN, embeddedMigrations)
	if err != nil {
		return fmt.Errorf("setting up db connections: %w", err)
	}

	cl := newClient(cr, pool)

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
	if err := cl.populateAppState(ctx); err != nil {
		return fmt.Errorf("checking app state values: %w", err)
	}
	setLogLevel(cl.Config.Debug)

	if err := cl.bootstrapAdmin(ctx); err != nil {
		return fmt.Errorf("bootstrapping initial admin key: %w", err)
	}

	if !mocking {
		if err := cl.initAllHooks(); err != nil {
			return fmt.Errorf("initializing webhooks: %w", err)
		}
	} else {
		slog.Debug("not initializing hooks since we are in mock mode")
	}

	return nil
}

func newClient(cr *creds, pool *pgxpool.Pool) *Client {
	slog.Debug("initializing server client")

	cwCreds := &psa.Creds{
		PublicKey:  cr.CWPublicKey,
		PrivateKey: cr.CWPrivateKey,
		ClientId:   cr.CWClientID,
		CompanyId:  cr.CWCompanyID,
	}

	cl := &Client{
		State:       defaultAppState,
		Config:      defaultAppConfig,
		Creds:       cr,
		Queries:     db.New(pool),
		Pool:        pool,
		CWClient:    psa.NewClient(cwCreds),
		WebexClient: webex.NewClient(cr.WebexSecret),
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
	vals := map[string]string{
		"ROOT_URL":            c.RootURL,
		"INITIAL_ADMIN_EMAIL": c.InitialAdminEmail,
		"POSTGRES_DSN":        c.PostgresDSN,
		"CW_PUB_KEY":          c.CWPublicKey,
		"CW_PRIV_KEY":         c.CWPrivateKey,
		"CW_CLIENT_ID":        c.CWClientID,
		"CW_COMPANY_ID":       c.CWCompanyID,
		"WEBEX_SECRET":        c.WebexSecret,
	}

	var empty []string
	for k, v := range vals {
		if v == "" {
			empty = append(empty, k)
		}
	}

	if len(empty) > 0 {
		return fmt.Errorf("1 or more required env variables are empty: %v", empty)
	}

	return nil
}
