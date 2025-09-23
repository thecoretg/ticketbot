package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
	"github.com/thecoretg/ticketbot/webex"

	"github.com/gin-gonic/gin"
)

type Server struct {
	Queries     *db.Queries
	GinEngine   *gin.Engine
	Config      *Cfg
	CWClient    *connectwise.Client
	WebexClient *webex.Client

	cwCompanyID string
	ticketLocks sync.Map
}

// InitAndRun initializes/verifies the config, bootstraps the initial admin/API key, checks and/or initializes webhooks, and runs the server.
func InitAndRun(ctx context.Context) error {
	cfg, err := InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	dbConn, err := ConnectToDB(ctx, cfg.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connecting to db: %w", err)
	}

	s := NewServer(cfg, dbConn)

	if err := s.BootstrapAdmin(context.Background()); err != nil {
		return fmt.Errorf("boostrapping admin: %w", err)
	}

	slog.Info("initializing webhooks")
	if err := s.InitAllHooks(); err != nil {
		return fmt.Errorf("initiating webhooks: %w", err)
	}

	if err := s.Run(); err != nil {
		return fmt.Errorf("running server: %w", err)
	}

	return nil
}

// Run just runs the server, and does not do the initialization steps. Good if it went down and you just need to
// restart it
func (s *Server) Run() error {
	if !s.Config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	return s.GinEngine.Run()
}

func (s *Server) addAllRoutes() {
	s.addHooksGroup()
	s.addBoardsGroup()
}

func ConnectToDB(ctx context.Context, dsn string) (*db.Queries, error) {
	dbConn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return db.New(dbConn), nil
}

func NewServer(cfg *Cfg, dbConn *db.Queries) *Server {
	cwCreds := &connectwise.Creds{
		PublicKey:  cfg.CwPubKey,
		PrivateKey: cfg.CwPrivKey,
		ClientId:   cfg.CwClientID,
		CompanyId:  cfg.CwCompanyID,
	}

	s := &Server{
		Config:      cfg,
		CWClient:    connectwise.NewClient(cwCreds),
		WebexClient: webex.NewClient(cfg.WebexSecret),
		Queries:     dbConn,
		GinEngine:   gin.Default(),
	}

	s.addAllRoutes()

	return s
}
