package ticketbot

import (
	"context"
	"fmt"
	"github.com/gin-gonic/autotls"
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
	cwCompanyID string
	webexClient *webex.Client
	ticketLocks sync.Map
	ginEngine   *gin.Engine
}

func RunServer() error {
	ctx := context.Background()
	config, err := InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	if err := setLogger(config.Debug, config.LogToFile, config.LogFilePath); err != nil {
		return fmt.Errorf("error setting logger: %w", err)
	}
	slog.Debug("DEBUG ON") // only prints if debug is on...so clever

	opClient, err := new1PasswordClient(ctx, config.OPSvcToken)
	if err != nil {
		return fmt.Errorf("creating 1password client: %w", err)
	}

	allCreds, err := getCreds(ctx, opClient)
	if err != nil {
		return fmt.Errorf("getting credentials from 1password")
	}

	store, err := createStore(allCreds.postgresDSN)
	if err != nil {
		return fmt.Errorf("creating store: %w", err)
	}

	s, err := newServer(allCreds, config, store)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	if err := s.prep(ctx, s.config.PreloadBoards, s.config.PreloadTickets); err != nil {
		return fmt.Errorf("preparing server: %w", err)
	}

	s.addAllRoutes()

	return s.run()
}

func (s *server) run() error {
	if s.config.UseAutocert {
		slog.Info("running server with auto TLS", "url", s.config.RootURL)
		return autotls.Run(s.ginEngine, s.config.RootURL)
	}

	return s.ginEngine.Run()
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

func newServer(creds *creds, cfg *Cfg, store Store) (*server, error) {

	cwCreds := &connectwise.Creds{
		PublicKey:  creds.cwPubKey,
		PrivateKey: creds.cwPrivKey,
		ClientId:   creds.cwClientID,
		CompanyId:  creds.cwCompanyID,
	}

	return &server{
		config:      cfg,
		cwClient:    connectwise.NewClient(cwCreds),
		cwCompanyID: creds.cwCompanyID,
		webexClient: webex.NewClient(http.DefaultClient, creds.webexSecret),
		dataStore:   store,
		ginEngine:   gin.Default(),
	}, nil
}

// createStore creates an in-memory store, or attempts to connect to a Postgres store if in the config.
func createStore(dsn string) (Store, error) {
	var store Store
	store = NewInMemoryStore()
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
