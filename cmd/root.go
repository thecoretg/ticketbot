package cmd

import (
	"context"
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/cfg"
	"github.com/thecoretg/ticketbot/internal/server"
	"github.com/thecoretg/ticketbot/internal/service"
)

var (
	ctx                                                = context.Background()
	configPath                                         string
	serve, preloadBoards, preloadTickets, initWebhooks bool
	maxPreloads                                        int
	rootCmd                                            = &cobra.Command{
		Use: "tbot",
	}

	runCmd = &cobra.Command{
		Use:  "run",
		RunE: runServer,
	}

	installServiceCmd = &cobra.Command{
		Use:  "install-service",
		RunE: runInstallService,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(installServiceCmd)
	rootCmd.PersistentFlags().BoolVarP(&preloadBoards, "preload-boards", "b", false, "preload boards from connectwise")
	rootCmd.PersistentFlags().BoolVarP(&preloadTickets, "preload-tickets", "t", false, "preload open tickets from connectwise")
	rootCmd.PersistentFlags().IntVarP(&maxPreloads, "max-preloads", "m", 5, "max simultaneous connectwise preloads")
	rootCmd.PersistentFlags().BoolVarP(&initWebhooks, "init-webhooks", "w", false, "initialize webhooks")
	rootCmd.PersistentFlags().BoolVarP(&serve, "serve", "s", false, "run the server")
	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "specify a config file path, otherwise defaults to $HOME/.config/ticketbot")
}

func runServer(cmd *cobra.Command, args []string) error {
	c, err := cfg.InitCfg(configPath)
	if err != nil {
		return fmt.Errorf("initializing config: %w", err)
	}

	d, err := server.ConnectToDB(ctx, c.Creds.PostgresDSN)
	if err != nil {
		return fmt.Errorf("connecting to database: %w", err)
	}

	s := server.NewServer(c, d)

	if preloadBoards {
		if err := s.PreloadBoards(ctx, maxPreloads); err != nil {
			return fmt.Errorf("preloading boards: %w", err)
		}
	}

	if preloadTickets {
		if err := s.PreloadOpenTickets(ctx, maxPreloads); err != nil {
			return fmt.Errorf("preloading tickets: %w", err)
		}
	}

	if initWebhooks {
		if err := s.InitAllHooks(); err != nil {
			return fmt.Errorf("initializing webhooks: %w", err)
		}
	}

	if serve {
		return s.Run(ctx)
	}

	return nil
}

func runInstallService(cmd *cobra.Command, args []string) error {
	if configPath == "" {
		return errors.New("config path is empty, please specify with --config or -c")
	}

	// initialize the config to just to validate it
	_, err := cfg.InitCfg(configPath)
	if err != nil {
		return fmt.Errorf("checking config: %w", err)
	}

	return service.Install(configPath)
}
