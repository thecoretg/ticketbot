package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	webexRoomsCmd = &cobra.Command{
		Use: "rooms",
	}

	syncRoomsCmd = &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.SyncRooms(); err != nil {
				return fmt.Errorf("syncing webex rooms: %w", err)
			}

			fmt.Println("Webex rooms sync started. You will not get confirmation, but this is usually done in less than a second.")
			return nil
		},
	}

	listWebexRoomsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			rooms, err := client.ListRooms()
			if err != nil {
				return fmt.Errorf("retrieving rooms from ticketbot: %w", err)
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
