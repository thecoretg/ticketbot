package cli

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/db"
	"github.com/thecoretg/ticketbot/ticketbot"
)

var (
	ctx     = context.Background()
	cfg     *ticketbot.Cfg
	server  *ticketbot.Server
	queries *db.Queries

	maxConcurrent  int
	initWebhooks   bool
	preloadBoards  bool
	preloadTickets bool
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

	preloadCmd.PersistentFlags().IntVarP(&maxConcurrent, "max-concurrent", "m", 5, "how many tickets to load at once")
	preloadCmd.PersistentFlags().BoolVarP(&preloadBoards, "boards", "b", false, "preload boards")
	preloadCmd.PersistentFlags().BoolVarP(&preloadTickets, "tickets", "t", false, "preload tickets")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(dbCmd)
	rootCmd.AddCommand(webexCmd)
	serverCmd.AddCommand(preloadCmd)
	serverCmd.AddCommand(initHooksCmd)
	serverCmd.AddCommand(runCmd)
	dbCmd.AddCommand(listBoardsCmd)
	webexCmd.AddCommand(listRoomsCmd)
}
