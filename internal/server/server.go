package server

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gin-gonic/autotls"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/internal/cfg"
	"github.com/thecoretg/ticketbot/internal/db"
	"github.com/thecoretg/ticketbot/internal/psa"
	"github.com/thecoretg/ticketbot/internal/webex"
)

type Client struct {
	Config      *cfg.Cfg
	State       *appState
	CWClient    *psa.Client
	WebexClient *webex.Client
	Pool        *pgxpool.Pool
	Queries     *db.Queries
	Server      *gin.Engine

	ticketLocks sync.Map
}

func Run(embeddedMigrations embed.FS) error {
	ctx := context.Background()
	c, err := cfg.InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	pool, err := setupDB(ctx, c.PostgresDSN, embeddedMigrations)
	s := newClient(c, pool)

	if err := s.startup(ctx); err != nil {
		return fmt.Errorf("running server startup: %w", err)
	}

	if err := s.serve(); err != nil {
		return fmt.Errorf("serving api: %w", err)
	}

	return nil
}

// Run just runs the server, and does not do the initialization steps. Good if it went down and you just need to
// restart it
func (cl *Client) serve() error {
	setInitialLogger()
	cl.Server = gin.Default()
	cl.addRoutes()

	if cl.Config.UseAutoTLS {
		slog.Debug("running server with auto tls", "url", cl.Config.RootURL)
		return autotls.Run(cl.Server, cl.Config.RootURL)
	}

	slog.Debug("running server without auto tls")
	return cl.Server.Run()
}

func (cl *Client) startup(ctx context.Context) error {
	if err := cl.populateAppState(ctx); err != nil {
		return fmt.Errorf("checking app state values: %w", err)
	}
	setLogLevel(cl.State.Debug)

	if err := cl.bootstrapAdmin(ctx); err != nil {
		return fmt.Errorf("bootstrapping initial admin key: %w", err)
	}

	if err := cl.initAllHooks(); err != nil {
		return fmt.Errorf("initializing webhooks: %w", err)
	}

	return nil
}

func newClient(cfg *cfg.Cfg, pool *pgxpool.Pool) *Client {
	slog.Debug("initializing server client")
	cwCreds := &psa.Creds{
		PublicKey:  cfg.CWPubKey,
		PrivateKey: cfg.CWPrivKey,
		ClientId:   cfg.CWClientID,
		CompanyId:  cfg.CWCompanyID,
	}

	s := &Client{
		Queries:     db.New(pool),
		Pool:        pool,
		Config:      cfg,
		CWClient:    psa.NewClient(cwCreds),
		State:       &appState{},
		WebexClient: webex.NewClient(cfg.WebexSecret),
	}

	return s
}
