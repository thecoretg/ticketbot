package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	webexRoomsCmd = &cobra.Command{
		Use: "recipients",
	}

	listWebexRoomsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			recips, err := client.ListRecipients()
			if err != nil {
				return err
			}

			if len(recips) == 0 {
				fmt.Println("No recipients found")
				return nil
			}

			recips = filterEmptyTitleRooms(recips)

			webexRoomsToTable(recips)
			return nil
		},
	}
)

func filterEmptyTitleRooms(rooms []models.WebexRecipient) []models.WebexRecipient {
	var f []models.WebexRecipient
	for _, r := range rooms {
		if r.Name != "Empty Title" {
			f = append(f, r)
		}
	}

	return f
}

func init() {
	webexRoomsCmd.AddCommand(listWebexRoomsCmd)
}
