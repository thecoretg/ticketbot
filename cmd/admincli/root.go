package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/sdk"
)

const (
	apiKeyEnv  = "TBOT_API_KEY"
	baseURLEnv = "TBOT_BASE_URL"
)

var (
	client  *sdk.Client
	rootCmd = &cobra.Command{
		Use: "tbot-admin",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			key := os.Getenv(apiKeyEnv)
			bu := os.Getenv(baseURLEnv)
			var err error
			client, err = sdk.NewClient(key, bu)
			if err != nil {
				return fmt.Errorf("creating api client: %w", err)
			}

			return client.TestConnection()
		},
	}

	getBoardsCmd = &cobra.Command{
		Use: "get-boards",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Connection successful")
			return nil
		},
	}
)

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(getBoardsCmd)
}
