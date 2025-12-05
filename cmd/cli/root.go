package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

var (
	boardID     int
	recipientID int

	client  *sdk.Client
	rootCmd = &cobra.Command{
		Use:               "tbot",
		PersistentPreRunE: createClient,
		SilenceUsage:      true,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(pingCmd, authCheckCmd, syncCmd, cfgCmd, webexRoomsCmd, cwBoardsCmd, notifiersCmd)
}
