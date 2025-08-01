package ticketbot

import (
	"tctg-automation/internal/ticketbot/cfg"
	"tctg-automation/internal/ticketbot/store"
	"testing"
)

func TestPrepServer(t *testing.T) {
	config, err := cfg.InitCfg()
	if err != nil {
		t.Fatalf("initializing config: %v", err)
	}

	if err := setLogger(config.Debug, config.LogToFile, config.LogFilePath); err != nil {
		t.Fatalf("error setting logger: %v", err)
	}

	s := newServer(config, store.NewInMemoryStore())
	if err := s.prep(true); err != nil {
		t.Fatalf("preparing server: %v", err)
	}
}
