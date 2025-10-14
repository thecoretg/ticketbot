package server

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

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

	cwCompanyID string
	ticketLocks sync.Map
}

// Run just runs the server, and does not do the initialization steps. Good if it went down and you just need to
// restart it
func (s *Server) Run() error {
	if !s.Config.Logging.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	s.GinEngine = gin.Default()
	s.addAllRoutes()
	return s.GinEngine.Run()
}

func (s *Server) addAllRoutes() {
	s.addHooksGroup()
	s.addBoardsGroup()
}

func ConnectToDB(ctx context.Context, dsn string) (*db.Queries, error) {
	slog.Info("connecting to postgres server")
	dbConn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return db.New(dbConn), nil
}

func NewServer(cfg *cfg.Cfg, dbConn *db.Queries) *Server {
	slog.Info("initializing server client")
	cwCreds := &psa.Creds{
		PublicKey:  cfg.Creds.CW.PubKey,
		PrivateKey: cfg.Creds.CW.PrivKey,
		ClientId:   cfg.Creds.CW.ClientID,
		CompanyId:  cfg.Creds.CW.CompanyID,
	}

	s := &Server{
		Config:      cfg,
		CWClient:    psa.NewClient(cwCreds),
		WebexClient: webex.NewClient(cfg.Creds.WebexSecret),
		Queries:     dbConn,
	}

	return s
}
