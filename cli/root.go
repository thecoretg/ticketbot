package cli

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var (
	ctx    = context.Background()
	cfg    *ticketbot.Cfg
	server *ticketbot.Server

	maxConcurrent  int
	initWebhooks   bool
	preloadBoards  bool
	preloadTickets bool
	listenAndServe bool

	rootCmd = &cobra.Command{
		Use:          "ticketbot-admin",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = ticketbot.InitCfg()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			server, err = ticketbot.NewServer(ctx, cfg, initWebhooks)
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

	listBoardsCmd = &cobra.Command{
		Use: "list-boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			boards, err := server.Queries.ListBoards(ctx)
			if err != nil {
				return fmt.Errorf("retrieving boards from database: %w", err)
			}

			if len(boards) > 0 {
				for _, b := range boards {
					fmt.Printf("%s ID:%d, Notify Enabled:%v\n", b.Name, b.ID, b.NotifyEnabled)
				}
			} else {
				fmt.Println("No boards were found in databas")
			}

			return nil
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&initWebhooks, "init-webhooks", "w", false, "initialize webhooks")
	rootCmd.PersistentFlags().BoolVarP(&listenAndServe, "run", "r", false, "run the full server")

	preloadCmd.PersistentFlags().IntVarP(&maxConcurrent, "max-concurrent", "m", 5, "how many tickets to load at once")
	preloadCmd.PersistentFlags().BoolVarP(&preloadBoards, "boards", "b", false, "preload boards")
	preloadCmd.PersistentFlags().BoolVarP(&preloadTickets, "tickets", "t", false, "preload tickets")

	rootCmd.AddCommand(preloadCmd)
	rootCmd.AddCommand(initHooksCmd)
	rootCmd.AddCommand(runCmd)
}
