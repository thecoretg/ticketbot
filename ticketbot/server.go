package ticketbot

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thecoretg/ticketbot/connectwise"
	"github.com/thecoretg/ticketbot/db"
	"github.com/thecoretg/ticketbot/webex"
	"log/slog"
	"sync"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config      *Cfg
	queries     *db.Queries
	cwClient    *connectwise.Client
	cwCompanyID string
	webexClient *webex.Client
	ticketLocks sync.Map
	ginEngine   *gin.Engine
}

func Run(ctx context.Context, initHooks, preloadBoards, preloadTickets bool, maxConcurrentPreloads int) error {
	config, err := InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	s, err := NewServer(ctx, config)
	if err != nil {
		return fmt.Errorf("initializing server struct: %w", err)
	}

	if initHooks {
		if err := s.initiateCWHooks(); err != nil {
			return fmt.Errorf("initiating connectwise webhooks: %w", err)
		}
	}

	if preloadBoards || preloadTickets {
		if err := s.PreloadData(ctx, preloadBoards, preloadTickets, maxConcurrentPreloads); err != nil {
			return fmt.Errorf("preloading data: %w", err)
		}
	}

	if !s.config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	s.addAllRoutes()
	return s.ginEngine.Run()
}

func GetGinEngine() (*gin.Engine, error) {
	ctx := context.Background()
	config, err := InitCfg()
	if err != nil {
		return nil, fmt.Errorf("initializing config: %w", err)
	}

	slog.Debug("DEBUG ON") // only prints if debug is on...so clever

	s, err := NewServer(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("creating Server: %w", err)
	}

	if err := s.initiateCWHooks(); err != nil {
		return nil, fmt.Errorf("initiating connectwise webhooks: %w", err)
	}
	if !s.config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	s.ginEngine = gin.Default()
	s.addAllRoutes()

	return s.ginEngine, nil
}

func (s *Server) addAllRoutes() {
	s.addHooksGroup()
	s.addBoardsGroup()
}

func NewServer(ctx context.Context, cfg *Cfg) (*Server, error) {
	dbConn, err := pgxpool.New(ctx, cfg.Creds.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

	q := db.New(dbConn)
	cwCreds := &connectwise.Creds{
		PublicKey:  cfg.Creds.CwPubKey,
		PrivateKey: cfg.Creds.CwPrivKey,
		ClientId:   cfg.Creds.CwClientID,
		CompanyId:  cfg.Creds.CwCompanyID,
	}

	return &Server{
		config:      cfg,
		cwClient:    connectwise.NewClient(cwCreds),
		webexClient: webex.NewClient(cfg.Creds.WebexSecret),
		queries:     q,
		ginEngine:   gin.Default(),
	}, nil
}
