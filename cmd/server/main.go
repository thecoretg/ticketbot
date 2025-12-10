package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/cmd/common"
	"github.com/thecoretg/ticketbot/internal/server"
)

func main() {
	if err := Run(); err != nil {
		fmt.Println("An error occured:", err)
	}
}

func Run() error {
	if len(os.Args[1:]) > 0 && slices.Contains([]string{"version", "v"}, os.Args[1]) {
		fmt.Println(common.ServerVersion)
		return nil
	}

	ctx := context.Background()

	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})))

	a, err := server.NewApp(ctx, common.GooseMigrationVersion)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
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
