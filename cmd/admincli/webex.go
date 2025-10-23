package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	roomsCmd = &cobra.Command{
		Use: "rooms",
	}

	listRoomsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			rooms, err := client.ListRooms(nil)
			if err != nil {
				return fmt.Errorf("listing rooms: %w", err)
			}

			for _, r := range rooms {
				fmt.Printf("%s | Type: %s  ID: %s\n", r.Title, r.Type, r.Id)
			}

			return nil
		},
	}
)

func addRoomsCmd() {
	rootCmd.AddCommand(roomsCmd)
	roomsCmd.AddCommand(listRoomsCmd)
}
