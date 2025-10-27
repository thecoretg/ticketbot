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

type Server struct {
	Queries     *db.Queries
	GinEngine   *gin.Engine
	Config      *cfg.Cfg
	CWClient    *psa.Client
	WebexClient *webex.Client

	State       *appState
	cwCompanyID string
	ticketLocks sync.Map
}

func Run(embeddedMigrations embed.FS) error {
	ctx := context.Background()
	c, err := cfg.InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	pool, err := pgxpool.New(ctx, c.PostgresDSN)
	if err != nil {
		return fmt.Errorf("creating pgx pool: %w", err)
	}

	if err := migrateDB(pool, embeddedMigrations); err != nil {
		return fmt.Errorf("connecting/migrating db: %w", err)
	}

	s := NewServer(c, pool)

	if err := s.populateAppState(ctx); err != nil {
		return fmt.Errorf("checking app state values: %w", err)
	}

	if err := s.checkAndRunInit(ctx); err != nil {
		return fmt.Errorf("running initialization: %w", err)
	}

	if err := s.serve(); err != nil {
		return fmt.Errorf("serving api: %w", err)
	}

	return nil
}

// Run just runs the server, and does not do the initialization steps. Good if it went down and you just need to
// restart it
func (s *Server) serve() error {
	s.GinEngine = gin.Default()
	s.addRoutes()

	if s.Config.UseAutoTLS {
		slog.Debug("running server with auto tls", "url", s.Config.RootURL)
		return autotls.Run(s.GinEngine, s.Config.RootURL)
	}

	slog.Debug("running server without auto tls")
	return s.GinEngine.Run()
}

func NewServer(cfg *cfg.Cfg, pool *pgxpool.Pool) *Server {
	slog.Debug("initializing server client")
	cwCreds := &psa.Creds{
		PublicKey:  cfg.CWPubKey,
		PrivateKey: cfg.CWPrivKey,
		ClientId:   cfg.CWClientID,
		CompanyId:  cfg.CWCompanyID,
	}

	s := &Server{
		Config:      cfg,
		CWClient:    psa.NewClient(cwCreds),
		WebexClient: webex.NewClient(cfg.WebexSecret),
		Queries:     db.New(pool),
	}

	return s
}
