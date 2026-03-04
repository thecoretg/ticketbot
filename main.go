package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/server"
)

const (
	gooseMigrationVersion = 5
	serverVersion         = "1.5.1"
	shutdownTimeout       = 10 * time.Second
)

func main() {
	if err := Run(); err != nil {
		fmt.Println("An error occured:", err)
		os.Exit(1)
	}
}

func Run() error {
	if len(os.Args[1:]) > 0 && slices.Contains([]string{"version", "v"}, os.Args[1]) {
		fmt.Println(serverVersion)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var level slog.LevelVar
	if os.Getenv("DEBUG") == "true" {
		level.Set(slog.LevelDebug)
	}

	var cwHandler *logging.CloudwatchHandler
	if logging.CloudwatchVarsSet() {
		var err error
		p := logging.GetCloudwatchParamsFromEnv(&level)
		cwHandler, err = logging.NewCloudwatchLogger(ctx, p)
		if err != nil {
			return fmt.Errorf("creating cloudwatch logger: %w", err)
		}
	}

	var baseLogger *slog.Logger
	if cwHandler != nil {
		slog.Info("using AWS log handler")
		baseLogger = slog.New(cwHandler)
	} else {
		slog.Info("using stdout json handler")
		baseLogger = logging.NewDefaultLogger(&level)
	}
	logBuf := logging.NewBufferHandler(baseLogger.Handler(), 500)
	logger := slog.New(logBuf)
	slog.SetDefault(logger)

	a, persister, err := server.NewApp(ctx, gooseMigrationVersion, &level, logBuf)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	logBuf.Resize(a.Config.LogBufferSize)
	if err := persister.SeedBuffer(ctx); err != nil {
		slog.Warn("failed to seed log buffer from db", "error", err)
	}
	persister.Start(ctx)

	if !a.TestFlags.SkipAuth {
		slog.Info("attempting to bootstrap admin")
		var adminPwd *string
		if a.Creds.InitialAdminPassword != "" {
			adminPwd = &a.Creds.InitialAdminPassword
		}
		if err := a.Svc.User.BootstrapAdmin(ctx, a.Creds.InitialAdminEmail, adminPwd); err != nil {
			return fmt.Errorf("bootstrapping admin: %w", err)
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
	srv.Use(gin.LoggerWithConfig(gin.LoggerConfig{Output: slogWriter}))
	srv.Use(gin.RecoveryWithWriter(slogWriter))
	server.AddRoutes(a, srv, cancel)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	httpSrv := &http.Server{
		Addr:    ":" + port,
		Handler: srv,
	}

	// listen for OS signals (SIGTERM from Docker, SIGINT from Ctrl+C)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case sig := <-sigCh:
			slog.Info("received signal, shutting down", "signal", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	// start serving
	go func() {
		slog.Info("server starting", "port", port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			cancel()
		}
	}()

	// block until context is cancelled (signal or restart request)
	<-ctx.Done()
	slog.Info("shutting down gracefully")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := httpSrv.Shutdown(shutdownCtx); err != nil {
		slog.Error("error during shutdown", "error", err)
	}

	return nil
}
