package server

import (
	"context"
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
	s.addRoutes()

	if s.Config.General.UseAutoTLS {
		slog.Debug("running server with auto tls", "url", s.Config.General.RootURL)
		return autotls.Run(s.GinEngine, s.Config.General.RootURL)
	}

	slog.Debug("running server without auto tls")
	return s.GinEngine.Run()
}

func ConnectToDB(ctx context.Context, dsn string) (*db.Queries, error) {
	slog.Debug("connecting to postgres server")
	dbConn, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("connecting to db: %w", err)
	}

	return db.New(dbConn), nil
}

func NewServer(cfg *cfg.Cfg, dbConn *db.Queries) *Server {
	slog.Debug("initializing server client")
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

func (s *Server) addRoutes() {
	ping := s.GinEngine.Group("/ping", ErrorHandler(s.Config.General.ExitOnError), s.APIKeyAuth())
	ping.GET("/", s.ping)

	boards := s.GinEngine.Group("/boards", ErrorHandler(s.Config.General.ExitOnError), s.APIKeyAuth())
	boards.GET("/:board_id", s.getBoard)
	boards.GET("/", s.listBoards)
	boards.PUT("/:board_id", s.putBoard)
	boards.DELETE("/:board_id", s.deleteBoard)

	rooms := s.GinEngine.Group("/rooms", ErrorHandler(s.Config.General.ExitOnError), s.APIKeyAuth())
	rooms.GET("/", s.listWebexRooms)

	hooks := s.GinEngine.Group("/hooks")
	cwHooks := hooks.Group("/cw", requireValidCWSignature(), ErrorHandler(s.Config.General.ExitOnError))
	cwHooks.POST("/tickets", s.handleTickets)
}
