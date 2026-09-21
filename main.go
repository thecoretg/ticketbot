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

	"github.com/thecoretg/tctg-go/entra"
	"github.com/thecoretg/ticketbot/internal/env"
	"github.com/thecoretg/ticketbot/internal/logging"
	"github.com/thecoretg/ticketbot/internal/middleware"
	"github.com/thecoretg/ticketbot/internal/server"
)

const (
	gooseMigrationVersion = 17
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

	// Workers start before the callback is registered so nothing arrives with no one to drain it.
	a.Svc.Intake.Start(ctx)

	if !e.SkipHooks {
		if err := a.Svc.Hooks.ProcessAllHooks(ctx); err != nil {
			return fmt.Errorf("processing connectwise hooks: %w", err)
		}
	}

	// SameOrigin rejects cross-site POST/PUT/DELETE by Origin header; webhooks carry no Origin
	// and pass through. Together with SameSite=Lax cookies this is the CSRF defence.
	handler := middleware.Chain(server.NewHandler(a, cancel),
		middleware.RequestLog(logger),
		middleware.Recover(logger),
		entra.SameOrigin,
	)

	httpSrv := &http.Server{
		Addr:              ":" + e.Port,
		Handler:           handler,
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
	a.Svc.Intake.Wait(shutdownCtx)

	return nil
}
