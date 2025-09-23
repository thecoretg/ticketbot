package cli

import (
	"context"

	"github.com/spf13/cobra"
)

var (
	ctx            = context.Background()
	initAndRun     bool
	initWebhooks   bool
	listenAndServe bool

	rootCmd = &cobra.Command{
		Use:          "ticketbot-admin",
		SilenceUsage: true,
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	serverCmd.PersistentFlags().BoolVarP(&initWebhooks, "init-webhooks", "w", false, "initialize webhooks")
	serverCmd.PersistentFlags().BoolVarP(&listenAndServe, "run", "r", false, "run the full server")

	runCmd.PersistentFlags().BoolVarP(&initAndRun, "init-and-run", "i", false, "initialize the server and run")

	rootCmd.AddCommand(serverCmd)
	serverCmd.AddCommand(runCmd)
}
