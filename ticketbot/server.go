package ticketbot

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
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

func GetGinEngine() (*gin.Engine, error) {
	ctx := context.Background()
	config, err := InitCfg(ctx)
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
	dbConn, err := pgx.Connect(ctx, cfg.Creds.PostgresDSN)
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
	}, nil
}
