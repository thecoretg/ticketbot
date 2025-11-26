package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	enableNotify     bool
	notifierID       int
	forwardID        int
	forwardSrcEmail  string
	forwardDestEmail string
	forwardStartDate string
	forwardEndDate   string
	forwardEnabled   bool
	forwardUserKeeps bool

	notifiersCmd = &cobra.Command{
		Use: "notifiers",
	}

	rulesCmd = &cobra.Command{
		Use: "rules",
	}

	forwardsCmd = &cobra.Command{
		Use: "forwards",
	}

	listNotifierRulesCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ns, err := client.ListNotifierRules()
			if err != nil {
				return err
			}

			if ns == nil || len(ns) == 0 {
				fmt.Println("No notifiers found")
				return nil
			}

			notifierRulesTable(ns)
			return nil
		},
	}

	getNotifierRuleCmd = &cobra.Command{
		Use: "get",
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := client.GetNotifierRule(notifierID)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nRoom: %d\nBoard: %d\nNotify: %v\n",
				n.ID, n.WebexRoomID, n.CwBoardID, n.NotifyEnabled)

			return nil
		},
	}

	createNotifierRuleCmd = &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			if boardID == 0 {
				return errors.New("board ID required")
			}

			if roomID == 0 {
				return errors.New("room ID required")
			}

			p := &models.Notifier{
				CwBoardID:     boardID,
				WebexRoomID:   roomID,
				NotifyEnabled: enableNotify,
			}

			n, err := client.CreateNotifierRule(p)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nRoom: %d\nBoard: %d\nNotify: %v\n",
				n.ID, n.WebexRoomID, n.CwBoardID, n.NotifyEnabled)

			return nil
		},
	}

	deleteNotifierRuleCmd = &cobra.Command{
		Use: "delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.DeleteNotifierRule(notifierID); err != nil {
				return err
			}

			fmt.Printf("Successfully deleted notifier with id of %d\n", notifierID)
			return nil
		},
	}

	listForwardsCmd = &cobra.Command{
		Use: "list",
		RunE: func(cmd *cobra.Command, args []string) error {
			ufs, err := client.ListUserForwards()
			if err != nil {
				return err
			}

			if ufs == nil || len(ufs) == 0 {
				fmt.Println("No user forwards found")
				return nil
			}

			userForwardsTable(ufs)
			return nil
		},
	}

	getForwardCmd = &cobra.Command{
		Use: "get",
		RunE: func(cmd *cobra.Command, args []string) error {
			uf, err := client.GetUserForward(notifierID)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nUser: %s\nForward To: %s\nStart Date: %s\nEnd Date: %s\n"+
				"User Keeps Copy: %v\nEnabled: %v\n",
				uf.ID, uf.UserEmail, uf.DestEmail, uf.StartDate, uf.EndDate, uf.UserKeepsCopy, uf.Enabled)

			return nil
		},
	}

	createForwardCmd = &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			if forwardSrcEmail == "" {
				return errors.New("source email required")
			}

			if forwardDestEmail == "" {
				return errors.New("destination email required")
			}

			if forwardStartDate == "" {
				return errors.New("start date required")
			}

			var start *time.Time
			st, err := time.Parse("2006-01-02", forwardStartDate)
			if err != nil {
				return fmt.Errorf("parsing start date: %w", err)
			}
			start = &st

			var end *time.Time
			if forwardEndDate != "" {
				et, err := time.Parse("2006-01-02", forwardEndDate)
				if err != nil {
					return err
				}
				end = &et
			}

			p := &models.UserForward{
				UserEmail:     forwardSrcEmail,
				DestEmail:     forwardDestEmail,
				StartDate:     start,
				EndDate:       end,
				Enabled:       forwardEnabled,
				UserKeepsCopy: forwardUserKeeps,
			}

			uf, err := client.CreateUserForward(p)
			if err != nil {
				return fmt.Errorf("creating user forward: %w", err)
			}

			fmt.Printf("ID: %d\nUser: %s\nForward To: %s\nStart Date: %s\nEnd Date: %s\nUser Keeps Copy: %v\nEnabled: %v\n",
				uf.ID, uf.UserEmail, uf.DestEmail, uf.StartDate, uf.EndDate, uf.UserKeepsCopy, uf.Enabled)

			return nil
		},
	}

	deleteForwardCmd = &cobra.Command{
		Use: "delete",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.DeleteUserForward(forwardID); err != nil {
				return err
			}

			fmt.Printf("Successfully deleted user forward with id of %d\n", forwardID)
			return nil
		},
	}
)

func init() {
	notifiersCmd.AddCommand(rulesCmd, forwardsCmd)
	rulesCmd.AddCommand(createNotifierRuleCmd, listNotifierRulesCmd, getNotifierRuleCmd, deleteNotifierRuleCmd)
	forwardsCmd.AddCommand(createForwardCmd, listForwardsCmd, getForwardCmd, deleteForwardCmd)
	rulesCmd.PersistentFlags().IntVar(&notifierID, "id", 0, "id of notifier")
	forwardsCmd.PersistentFlags().IntVar(&forwardID, "id", 0, "id of user forward")
	createNotifierRuleCmd.Flags().IntVarP(&boardID, "board-id", "b", 0, "board id to use")
	createNotifierRuleCmd.Flags().IntVarP(&roomID, "room-id", "r", 0, "room id to use")
	createNotifierRuleCmd.Flags().BoolVarP(&enableNotify, "enable", "x", false, "enable notify for rule")
	createForwardCmd.Flags().BoolVarP(&forwardUserKeeps, "user-keeps-copy", "k", false, "user keeps a copy of forwarded emails")
	createForwardCmd.Flags().StringVarP(&forwardSrcEmail, "source-email", "s", "", "source email address to forward from")
	createForwardCmd.Flags().StringVarP(&forwardDestEmail, "dest-email", "d", "", "destination email address to forward to")
	createForwardCmd.Flags().StringVarP(&forwardStartDate, "start-date", "a", "", "start date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().StringVarP(&forwardEndDate, "end-date", "e", "", "end date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().BoolVarP(&forwardEnabled, "enabled", "x", true, "enable the forward")
}
