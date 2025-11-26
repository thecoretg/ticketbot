package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/server"
)

func main() {
	if err := Run(); err != nil {
		fmt.Println("An error occured:", err)
	}
}

func Run() error {
	ctx := context.Background()
	a, err := server.NewApp(ctx)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}
	if a.Config.Debug {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("DEBUG ON")
	}

	if !a.TestFlags.SkipAuth {
		slog.Info("attempting to bootstrap admin")
		if err := a.Svc.User.BootstrapAdmin(ctx, a.Creds.InitialAdminEmail, a.TestFlags.APIKey); err != nil {
			return fmt.Errorf("bootstrapping admin api key: %w", err)
		}
	} else {
		slog.Info("SKIP AUTH ENABLED")
	}

	if !a.TestFlags.SkipHooks {
		if err := a.Svc.Hooks.ProcessCWHooks(); err != nil {
			return fmt.Errorf("processing connectwise hooks: %w", err)
		}
	}

	srv := gin.Default()
	server.AddRoutes(a, srv)

	return srv.Run()
}
