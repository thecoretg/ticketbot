package main

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	syncAll, syncBoards, syncRooms, syncTickets bool
	syncBoardIDs                                []int
	maxConcurrentSyncs                          int
	syncCmd                                     = &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			if syncAll {
				syncBoards = true
				syncRooms = true
				syncTickets = true
			}

			if !syncBoards && !syncRooms && !syncTickets {
				return errors.New("at least one sync target must be set")
			}

			if syncTickets && len(syncBoardIDs) == 0 {
				fmt.Println("WARNING: Ticket sync enabled, but no board IDs provided. All boards will be included; this may take a while.")
			}

			p := &models.SyncPayload{
				WebexRooms:         syncRooms,
				CWBoards:           syncBoards,
				CWTickets:          syncTickets,
				BoardIDs:           syncBoardIDs,
				MaxConcurrentSyncs: maxConcurrentSyncs,
			}
			if err := client.Sync(p); err != nil {
				return err
			}

			fmt.Println("Sync started. You will not get confirmation, but this is usually done in less than a second.")
			return nil
		},
	}
)

func init() {
	syncCmd.Flags().BoolVar(&syncAll, "all", false, "sync boards, rooms, and tickets (will sync all boards for tickets unless specified)")
	syncCmd.Flags().BoolVarP(&syncBoards, "boards", "b", false, "sync connectwise boards")
	syncCmd.Flags().BoolVarP(&syncRooms, "rooms", "r", false, "sync webex rooms")
	syncCmd.Flags().BoolVarP(&syncTickets, "tickets", "t", false, "sync connectwise tickets; this will take a while")
	syncCmd.Flags().IntSliceVarP(&syncBoardIDs, "sync-boards", "i", nil, "board ids to sync")
	syncCmd.Flags().IntVar(&maxConcurrentSyncs, "max-syncs", 5, "max amount of concurrent syncs to run")
}
