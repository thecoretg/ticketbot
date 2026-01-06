package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/cmd/common"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

var (
	client  *sdk.Client
	rootCmd = &cobra.Command{
		Use:          "tbot",
		SilenceUsage: true,
	}

	versionCmd = &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(common.ServerVersion)
		},
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(adminCmd, versionCmd, pingCmd, authCheckCmd, syncCmd, listCmd, getCmd, createCmd, updateCmd, deleteCmd)
}

func createClient(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	var err error
	key := os.Getenv("TBOT_API_KEY")
	base := os.Getenv("TBOT_BASE_URL")

	client, err = sdk.NewClient(key, base)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	return nil
}
