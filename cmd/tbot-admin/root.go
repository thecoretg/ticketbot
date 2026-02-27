package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/cmd/common"
	"github.com/thecoretg/ticketbot/internal/tui"
	"github.com/thecoretg/ticketbot/sdk"
)

var (
	client  *sdk.Client
	rootCmd = &cobra.Command{
		Use:          "tbot",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := createClient(cmd, args); err != nil {
				return fmt.Errorf("creating client: %w", err)
			}

			if err := client.AuthTest(); err != nil {
				return fmt.Errorf("error authenticating: %w", err)
			}

			m := tui.NewModel(client, currentAPIKey)
			if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
				return err
			}

			return nil
		},
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
	rootCmd.AddCommand(versionCmd, pingCmd, authCheckCmd, syncCmd, listCmd, getCmd, createCmd, updateCmd, deleteCmd)
}

var currentAPIKey string

func createClient(cmd *cobra.Command, args []string) error {
	_ = godotenv.Load()

	var err error
	currentAPIKey = os.Getenv("TBOT_API_KEY")
	base := os.Getenv("TBOT_BASE_URL")

	client, err = sdk.NewClient(currentAPIKey, base)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	return nil
}
