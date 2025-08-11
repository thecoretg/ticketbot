package main

import (
	"context"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/ticketbot"
	"log"
)

var (
	ctx            = context.Background()
	server         *ticketbot.Server
	maxConcurrent  int
	initWebhooks   bool
	preloadBoards  bool
	preloadTickets bool

	rootCmd = &cobra.Command{
		Use: "ticketbot-admin",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			return ticketbot.Run(ctx, initWebhooks, preloadBoards, preloadTickets, maxConcurrent)
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
	rootCmd.PersistentFlags().IntVarP(&maxConcurrent, "max-concurrent", "m", 5, "how many tickets to load at once")
	rootCmd.PersistentFlags().BoolVarP(&preloadBoards, "preload-boards", "b", false, "preload boards")
	rootCmd.PersistentFlags().BoolVarP(&preloadTickets, "preload-tickets", "t", false, "preload tickets")
	rootCmd.PersistentFlags().BoolVarP(&initWebhooks, "init-webhooks", "w", false, "initialize webhooks")
}
