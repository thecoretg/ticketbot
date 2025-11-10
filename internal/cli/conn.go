package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	pingCmd = &cobra.Command{
		Use: "ping",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.Ping(); err != nil {
				return fmt.Errorf("testing server connection: %w", err)
			}

			fmt.Println("Server is up")
			return nil
		},
	}

	authCheckCmd = &cobra.Command{
		Use: "authcheck",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.AuthTest(); err != nil {
				return fmt.Errorf("testing api creds: %w", err)
			}

			return nil
		},
	}
)
