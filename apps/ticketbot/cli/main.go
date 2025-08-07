package main

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"log"
	"log/slog"
	"tctg-automation/internal/ticketbot"
)

var (
	ctx            = context.Background()
	server         *ticketbot.Server
	maxConcurrent  int
	preloadBoards  bool
	preloadTickets bool

	rootCmd = &cobra.Command{
		Use: "ticketbot-admin",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			config, err := ticketbot.InitCfg(ctx)
			if err != nil {
				return fmt.Errorf("initializing config: %w", err)
			}

			slog.Debug("DEBUG ON") // only prints if debug is on...so clever

			server, err = ticketbot.NewServer(config)
			if err != nil {
				return fmt.Errorf("creating Server: %w", err)
			}

			return nil
		},
	}

	preloadCmd = &cobra.Command{
		Use: "preload",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.PreloadData(ctx, preloadBoards, preloadTickets, maxConcurrent)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		log.Fatalf(err.Error())
	}
}

func init() {
	rootCmd.AddCommand(preloadCmd)
	preloadCmd.PersistentFlags().IntVarP(&maxConcurrent, "max-concurrent", "m", 5, "how many tickets to load at once")
	preloadCmd.PersistentFlags().BoolVarP(&preloadBoards, "preload-boards", "b", false, "preload boards")
	preloadCmd.PersistentFlags().BoolVarP(&preloadTickets, "preload-tickets", "t", false, "preload tickets")
}
