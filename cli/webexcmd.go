package cli

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/ticketbot"
	"os"
	"slices"
)

var (
	webexCmd = &cobra.Command{
		Use: "webex",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			cfg, err = ticketbot.InitCfg()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			server, err = ticketbot.NewServer(ctx, cfg, initWebhooks)
			if err != nil {
				return fmt.Errorf("starting server: %w", err)
			}
			return nil
		},
	}

	listRoomsCmd = &cobra.Command{
		Use: "list-rooms",
		RunE: func(cmd *cobra.Command, args []string) error {
			rooms, err := server.WebexClient.ListRooms(nil)
			if err != nil {
				return fmt.Errorf("getting rooms: %w", err)
			}

			if len(rooms) == 0 {
				fmt.Println("No rooms were found")
				os.Exit(0)
			}

			var sorted []string
			for _, r := range rooms {
				s := fmt.Sprintf("%s: %s", r.Title, r.Id)
				sorted = append(sorted, s)
			}

			slices.Sort(sorted)
			for _, r := range sorted {
				fmt.Println(r)
			}

			return nil
		},
	}
)
