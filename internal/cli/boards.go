package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cwBoardsCmd = &cobra.Command{
		Use: "boards",
	}

	syncBoardsCmd = &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.SyncBoards(); err != nil {
				return fmt.Errorf("syncing boards: %w", err)
			}

			fmt.Println("Boards sync started. You will not get confirmation, but this is usually done in less than a second.")
			return nil
		},
	}

	listBoardsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			boards, err := client.ListBoards()
			if err != nil {
				return fmt.Errorf("retrieving boards from ticketbot: %w", err)
			}

			if boards == nil || len(boards) == 0 {
				fmt.Println("No boards found")
				return nil
			}

			cwBoardsToTable(boards)
			return nil
		},
	}
)

func init() {
	cwBoardsCmd.AddCommand(syncBoardsCmd, listBoardsCmd)
}
