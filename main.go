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
	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/server"
)

const (
	gooseMigrationVersion = 11
	shutdownTimeout       = 10 * time.Second

	// HTTP server timeouts. Webhook and dashboard requests are small and fast; anything slower is
	// a stuck client holding a connection.
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 15 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
)

var serverVersion = "dev"

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

	e, err := env.Load()
	if err != nil {
		return fmt.Errorf("loading environment: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var level slog.LevelVar
	if e.Debug {
		level.Set(slog.LevelDebug)
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	baseLogger := logging.NewDefaultLogger(&level)
	logBuf := logging.NewBufferHandler(baseLogger.Handler(), 500)
	logger := slog.New(logBuf)
	slog.SetDefault(logger)

	a, persister, err := server.NewApp(ctx, e, gooseMigrationVersion, &level, logBuf)
	if err != nil {
		return fmt.Errorf("initializing app: %w", err)
	}

	logBuf.Resize(a.Config.LogBufferSize)
	if err := persister.SeedBuffer(ctx); err != nil {
		slog.Warn("failed to seed log buffer from db", "error", err)
	}
	persister.Start(ctx)

	slog.Info("attempting to bootstrap admin")
	var adminPwd *string
	if e.InitialAdminPassword != "" {
		adminPwd = &e.InitialAdminPassword
	}
	if err := a.Svc.User.BootstrapAdmin(ctx, e.InitialAdminEmail, adminPwd); err != nil {
		return fmt.Errorf("bootstrapping admin: %w", err)
	}

	if !e.SkipHooks {
		if err := a.Svc.Hooks.ProcessAllHooks(ctx); err != nil {
			return fmt.Errorf("processing connectwise hooks: %w", err)
		}
	}

	srv := gin.New()
	slogWriter := middleware.NewSlogWriter(logger)
	srv.Use(gin.LoggerWithConfig(gin.LoggerConfig{Output: slogWriter}))
	srv.Use(gin.RecoveryWithWriter(slogWriter))
	server.AddRoutes(a, srv, cancel)

	httpSrv := &http.Server{
		Addr:              ":" + e.Port,
		Handler:           srv,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
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
		slog.Info("server starting", "port", e.Port, "version", serverVersion)
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
