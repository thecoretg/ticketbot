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

func NewServer(ctx context.Context, cfg *Cfg, initHooks bool) (*Server, error) {
	slog.Info("beginning server initialization")
	dbConn, err := pgxpool.New(ctx, cfg.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("connecting to database: %w", err)
	}

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
		Queries:     db.New(dbConn),
		GinEngine:   gin.Default(),
	}

	if initHooks {
		slog.Info("initializing webhooks")
		if err := s.InitAllHooks(); err != nil {
			return nil, fmt.Errorf("initiating webhooks: %w", err)
		}
	}

	s.addAllRoutes()

	return s, nil
}
