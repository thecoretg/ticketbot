package ticketbot

import (
	"fmt"
	"log/slog"
	"net/http"
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
}

func Run() error {
	config, err := cfg.InitCfg()
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	if err := setLogger(config.Debug, config.LogToFile); err != nil {
		return fmt.Errorf("error setting logger: %w", err)
	}

	slog.Debug("DEBUG ON")

	s := &server{
		config:      config,
		cwClient:    connectwise.NewClient(&config.CWCreds),
		webexClient: webex.NewClient(http.DefaultClient, config.WebexBotSecret),
		dataStore:   store.NewInMemoryStore(),
	}

	if err := s.initiateCWHooks(); err != nil {
		return fmt.Errorf("initiating connectwise hooks: %w", err)
	}

	r := gin.Default()
	s.addTicketGroup(r)

	if err := r.Run(":80"); err != nil {
		return fmt.Errorf("error running server: %w", err)
	}

	return nil
}
