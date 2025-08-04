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

type server struct {
	config      *Cfg
	dataStore   Store
	cwClient    *connectwise.Client
	webexClient *webex.Client
	ticketLocks sync.Map
	ginEngine   *gin.Engine
}

func Run() error {
	ctx := context.Background()
	config, err := InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	if err := setLogger(config.Debug, config.LogToFile, config.LogFilePath); err != nil {
		return fmt.Errorf("error setting logger: %w", err)
	}
	slog.Debug("DEBUG ON") // only prints if debug is on...so clever

	store, err := createStore(config)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	s := newServer(config, store)
	if err := s.prep(ctx, true, true); err != nil {
		return fmt.Errorf("preparing server: %w", err)
	}

	s.addAllRoutes()

	if err := s.ginEngine.Run(":80"); err != nil {
		return fmt.Errorf("error running server: %w", err)
	}
	return nil
}

func (s *server) prep(ctx context.Context, preloadBoards, preloadTickets bool) error {
	if err := s.initiateCWHooks(); err != nil {
		return fmt.Errorf("initiating connectwise webhooks: %w", err)
	}

	if preloadBoards || preloadTickets {
		if err := s.preloadFromConnectwise(ctx, preloadBoards, preloadTickets); err != nil {
			return fmt.Errorf("preloading from connectwise: %w", err)
		}
	}

	return nil
}

func (s *server) addAllRoutes() {
	s.addHooksGroup()
	s.addTicketsGroup()
	s.addBoardsGroup()
}

func newServer(cfg *Cfg, store Store) *server {
	cwCreds := &connectwise.Creds{
		PublicKey:  cfg.CWPublicKey,
		PrivateKey: cfg.CWPrivateKey,
		ClientId:   cfg.CWClientID,
		CompanyId:  cfg.CWCompanyID,
	}

	return &server{
		config:      cfg,
		cwClient:    connectwise.NewClient(cwCreds),
		webexClient: webex.NewClient(http.DefaultClient, cfg.WebexBotSecret),
		dataStore:   store,
		ginEngine:   gin.Default(),
	}
}

// createStore creates an in-memory store, or attempts to connect to a Postgres store if in the config.
func createStore(cfg *Cfg) (Store, error) {
	var store Store
	store = NewInMemoryStore()
	if cfg.PostgresDSN != "" {
		slog.Debug("database connection string provided in config")

		var err error
		store, err = NewPostgresStore(cfg.PostgresDSN)
		if err != nil {
			return nil, fmt.Errorf("creating postgres store: %w", err)
		}
	}

	return store, nil
}
