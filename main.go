package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/cmd/common"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/models"
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

	var cwHandler *logging.CloudwatchHandler
	if logging.CloudwatchVarsSet() {
		var err error
		p := logging.GetCloudwatchParamsFromEnv()
		cwHandler, err = logging.NewCloudwatchLogger(ctx, p)
		if err != nil {
			return fmt.Errorf("creating cloudwatch logger: %w", err)
		}
	}

	var logger *slog.Logger
	if cwHandler != nil {
		slog.Info("using AWS log handler")
		logger = slog.New(cwHandler)
	} else {
		slog.Info("using stdout json handler")
		logger = logging.NewDefaultLogger(level)
	}
	slog.SetDefault(logger)

	a, err := server.NewApp(ctx, common.GooseMigrationVersion)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	if !a.Config.SkipLaunchSyncs {
		slog.Info("syncing webex rooms and connectwise boards")
		p := &models.SyncPayload{
			WebexRecipients:    true,
			CWBoards:           true,
			MaxConcurrentSyncs: 10,
		}

		if err := a.Svc.Sync.Sync(ctx, p); err != nil {
			// just log but continue
			slog.Error("error syncing webex recipients and connectwise boards", "error", err.Error())
		}
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
		if err := a.Svc.Hooks.ProcessAllHooks(); err != nil {
			return fmt.Errorf("processing connectwise hooks: %w", err)
		}
	}

	srv := gin.New()
	slogWriter := middleware.NewSlogWriter(logger)

	// send both logs and recovery to the logger, which is likely cloudwatch
	srv.Use(gin.LoggerWithConfig(gin.LoggerConfig{Output: slogWriter}))
	srv.Use(gin.RecoveryWithWriter(slogWriter))
	server.AddRoutes(a, srv)

	return srv.Run()
}
