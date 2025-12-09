package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	listCmd = &cobra.Command{
		Use: "list",
	}

	listBoardsCmd = &cobra.Command{
		Use: "boards",
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

	listNotifierRulesCmd = &cobra.Command{
		Use:     "notifier-rules",
		Aliases: []string{"rules"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, err := client.ListNotifierRules()
			if err != nil {
				return err
			}

			if len(ns) == 0 {
				fmt.Println("No notifiers found")
				return nil
			}

			notifierRulesTable(ns)
			return nil
		},
	}

	listForwardsCmd = &cobra.Command{
		Use:     "notifier-forwards",
		Aliases: []string{"forwards", "fwds"},
		RunE: func(cmd *cobra.Command, args []string) error {
			ufs, err := client.ListUserForwards()
			if err != nil {
				return err
			}

			if len(ufs) == 0 {
				fmt.Println("No user forwards found")
				return nil
			}

			userForwardsTable(ufs)
			return nil
		},
	}

	listWebexRecipientsCmd = &cobra.Command{
		Use: "recipients",
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

	listUsersCmd = &cobra.Command{
		Use: "users",
		RunE: func(cmd *cobra.Command, args []string) error {
			users, err := client.ListUsers()
			if err != nil {
				return err
			}

			if len(users) == 0 {
				fmt.Println("No users found")
				return nil
			}

			apiUsersTable(users)
			return nil
		},
	}

	listAPIKeysCmd = &cobra.Command{
		Use:     "api-keys",
		Aliases: []string{"keys"},
		RunE: func(cmd *cobra.Command, args []string) error {
			keys, err := client.ListAPIKeys()
			if err != nil {
				return err
			}

			if len(keys) == 0 {
				fmt.Println("No keys found")
				return nil
			}

			apiKeysTable(keys)
			return nil
		},
	}
)

func init() {
	listCmd.AddCommand(listBoardsCmd, listNotifierRulesCmd, listForwardsCmd,
		listWebexRecipientsCmd, listUsersCmd, listAPIKeysCmd)
}

func filterEmptyTitleRooms(rooms []models.WebexRecipient) []models.WebexRecipient {
	var f []models.WebexRecipient
	for _, r := range rooms {
		if r.Name != "Empty Title" {
			f = append(f, r)
		}
	}

	return f
}
