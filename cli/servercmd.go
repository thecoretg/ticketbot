package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var (
	serverCmd = &cobra.Command{
		Use: "server",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = ticketbot.InitCfg()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			server, err = ticketbot.NewServer(ctx, cfg, initWebhooks)
			if err != nil {
				return fmt.Errorf("starting server: %w", err)
			}
			return nil
		},
	}

	initHooksCmd = &cobra.Command{
		Use: "init-hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.InitAllHooks()
		},
	}

	preloadCmd = &cobra.Command{
		Use: "preload",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.PreloadData(ctx, preloadBoards, preloadTickets, maxConcurrent)
		},
	}

	runCmd = &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			return server.Run()
		},
	}
)
