package ticketbot

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"tctg-automation/internal/ticketbot/cfg"
	"tctg-automation/internal/ticketbot/store"
	"tctg-automation/pkg/connectwise"
	"tctg-automation/pkg/webex"

	"github.com/gin-gonic/gin"
)

type server struct {
	config      *cfg.Cfg
	dataStore   store.Store
	cwClient    *connectwise.Client
	webexClient *webex.Client
	ticketLocks sync.Map
	ginEngine   *gin.Engine
}

func Run() error {
	config, err := cfg.InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	if err := setLogger(config.Debug, config.LogToFile, config.LogFilePath); err != nil {
		return fmt.Errorf("error setting logger: %w", err)
	}

	slog.Debug("DEBUG ON") // only prints if debug is on...so clever

	s := newServer(config, store.NewInMemoryStore())
	if err := s.prep(true); err != nil {
		return fmt.Errorf("preparing server: %w", err)
	}

	s.addAllRoutes()

	if err := s.ginEngine.Run(":80"); err != nil {
		return fmt.Errorf("error running server: %w", err)
	}
	return nil
}

func (s *server) prep(preloadOpenTickets bool) error {
	if err := s.initiateCWHooks(); err != nil {
		return fmt.Errorf("initiating connectwise webhooks: %w", err)
	}

	if preloadOpenTickets {
		if err := s.preloadOpenTickets(); err != nil {
			return fmt.Errorf("preloading existing open tickets: %w", err)
		}
	}

	return nil
}

func (s *server) addAllRoutes() {
	s.addHooksGroup()
	s.addTicketsGroup()
	s.addBoardsGroup()
}

func newServer(cfg *cfg.Cfg, store store.Store) *server {
	return &server{
		config:      cfg,
		cwClient:    connectwise.NewClient(&cfg.CWCreds),
		webexClient: webex.NewClient(http.DefaultClient, cfg.WebexBotSecret),
		dataStore:   store,
		ginEngine:   gin.Default(),
	}
}
