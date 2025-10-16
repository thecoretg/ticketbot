package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/cfg"
	"github.com/thecoretg/ticketbot/internal/server"
)

var (
	ctx         = context.Background()
	config      *cfg.Cfg
	srv         *server.Server
	configPath  string
	maxPreloads int
	rootCmd     = &cobra.Command{
		Use: "tbot",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			config, err = cfg.InitCfg(configPath)
			if err != nil {
				return fmt.Errorf("initializing config: %w", err)
			}

			d, err := server.ConnectToDB(ctx, config.Creds.PostgresDSN)
			if err != nil {
				return fmt.Errorf("connecting to database: %w", err)
			}

			srv = server.NewServer(config, d)
			return nil
		},
	}

	hooksCmd = &cobra.Command{
		Use:   "hooks",
		Short: "Initialize webhooks for the TicketBot server in Connectwise PSA",
		RunE: func(cmd *cobra.Command, args []string) error {
			return srv.InitAllHooks()
		},
	}

	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Run the server",
		RunE: func(cmd *cobra.Command, args []string) error {
			return srv.Run(ctx)
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(preloadCmd)
	rootCmd.AddCommand(hooksCmd)
	rootCmd.AddCommand(runCmd)
	addServiceCmd()
	addPreloadCmd()

	rootCmd.PersistentFlags().StringVarP(&configPath, "config", "c", "", "specify a config file path, otherwise defaults to $HOME/.config/ticketbot")
}
