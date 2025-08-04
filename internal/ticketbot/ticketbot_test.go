package ticketbot

import (
	"context"
	"testing"
)

func TestPrepServer(t *testing.T) {
	ctx := context.Background()
	config, err := InitCfg()
	if err != nil {
		t.Fatalf("initializing config: %v", err)
	}

	if err := setLogger(config.Debug, config.LogToFile, config.LogFilePath); err != nil {
		t.Fatalf("error setting logger: %v", err)
	}

	s := newServer(config, NewInMemoryStore())
	if err := s.prep(ctx, true, false); err != nil {
		t.Fatalf("preparing server: %v", err)
	}
}

func TestRunServer(t *testing.T) {
	if err := Run(); err != nil {
		t.Fatalf("error running server: %v", err)
	}
}
