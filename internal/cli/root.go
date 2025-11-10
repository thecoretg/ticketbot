package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/sdk"
)

var (
	client  *sdk.Client
	rootCmd = &cobra.Command{
		Use:               "tbot",
		PersistentPreRunE: cliPreRun,
	}

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

	cwBoardsCmd = &cobra.Command{
		Use: "boards",
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

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		return err
	}

	return nil
}

func init() {
	rootCmd.AddCommand(webexRoomsCmd, cwBoardsCmd)
	webexRoomsCmd.AddCommand(syncRoomsCmd, listWebexRoomsCmd)
	cwBoardsCmd.AddCommand(listBoardsCmd)
}

func cliPreRun(cmd *cobra.Command, args []string) error {
	var err error
	key := os.Getenv("TBOT_API_KEY")
	base := os.Getenv("TBOT_BASE_URL")

	if key == "" {
		return errors.New("api key is empty")
	}

	if base == "" {
		return errors.New("base url is empty")
	}

	client, err = sdk.NewClient(key, base)
	if err != nil {
		return fmt.Errorf("creating api client: %w", err)
	}

	if err := client.TestConnection(); err != nil {
		return fmt.Errorf("testing connection: %w", err)
	}

	return nil
}
