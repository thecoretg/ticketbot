package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	webexRoomsCmd = &cobra.Command{
		Use: "rooms",
	}

	listWebexRoomsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			rooms, err := client.ListRooms()
			if err != nil {
				return err
			}

			if rooms == nil || len(rooms) == 0 {
				fmt.Println("No rooms found")
				return nil
			}

			webexRoomsToTable(rooms)
			return nil
		},
	}
)

func init() {
	webexRoomsCmd.AddCommand(listWebexRoomsCmd)
}
