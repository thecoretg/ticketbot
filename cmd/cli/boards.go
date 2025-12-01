package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cwBoardsCmd = &cobra.Command{
		Use: "boards",
	}

	listBoardsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			boards, err := client.ListBoards()
			if err != nil {
				return err
			}

			if len(boards) == 0 {
				fmt.Println("No boards found")
				return nil
			}

			cwBoardsToTable(boards)
			return nil
		},
	}
)

func init() {
	cwBoardsCmd.AddCommand(listBoardsCmd)
}
