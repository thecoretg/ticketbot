package ticketbot

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config      *Cfg
	dataStore   Store
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

	s, err := NewServer(config)
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

func NewServer(cfg *Cfg) (*Server, error) {
	store, err := createStore(cfg.Creds.PostgresDSN)
	if err != nil {
		return nil, fmt.Errorf("setting up postgres connection: %w", err)
	}

	cwCreds := &connectwise.Creds{
		PublicKey:  cfg.Creds.CwPubKey,
		PrivateKey: cfg.Creds.CwPrivKey,
		ClientId:   cfg.Creds.CwClientID,
		CompanyId:  cfg.Creds.CwCompanyID,
	}

	return &Server{
		config:      cfg,
		cwClient:    connectwise.NewClient(cwCreds),
		webexClient: webex.NewClient(http.DefaultClient, cfg.Creds.WebexSecret),
		dataStore:   store,
	}, nil
}

// createStore creates an in-memory store, or attempts to connect to a Postgres store if in the config.
func createStore(dsn string) (Store, error) {
	var store Store
	//store = NewInMemoryStore()
	if dsn != "" {
		slog.Debug("database connection string provided in config")

		var err error
		store, err = NewPostgresStore(dsn)
		if err != nil {
			return nil, fmt.Errorf("creating postgres store: %w", err)
		}
	}

	return store, nil
}
