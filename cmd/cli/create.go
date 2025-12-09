package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	createCmd = &cobra.Command{
		Use:     "create",
		Aliases: []string{"add"},
	}

	createNotifierRuleCmd = &cobra.Command{
		Use:     "notifier-rule",
		Aliases: []string{"rule"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var (
				p   *models.NotifierRule
				err error
			)

			if boardID == 0 {
				return errors.New("board is is required")
			}

			if recipientID == 0 {
				return errors.New("recipient id is required")
			}

			p = &models.NotifierRule{
				CwBoardID:        boardID,
				WebexRecipientID: recipientID,
				NotifyEnabled:    true,
			}

			n, err := client.CreateNotifierRule(p)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nRoom: %d\nBoard: %d\nNotify: %v\n",
				n.ID, n.WebexRecipientID, n.CwBoardID, n.NotifyEnabled)

			return nil
		},
	}

	createForwardCmd = &cobra.Command{
		Use:     "notifier-forward",
		Aliases: []string{"forward", "fwd"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if forwardSrcID == 0 {
				return errors.New("source email required")
			}

			if forwardDestID == 0 {
				return errors.New("destination email required")
			}

			var start *time.Time
			var st time.Time
			if forwardStartDate == "" {
				st = time.Now()
			} else {
				var err error
				st, err = time.Parse("2006-01-02", forwardStartDate)
				if err != nil {
					return fmt.Errorf("parsing start date: %w", err)
				}
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

			p := &models.NotifierForward{
				SourceID:      forwardSrcID,
				DestID:        forwardDestID,
				StartDate:     start,
				EndDate:       end,
				Enabled:       forwardEnabled,
				UserKeepsCopy: forwardUserKeeps,
			}

			uf, err := client.CreateUserForward(p)
			if err != nil {
				return fmt.Errorf("creating user forward: %w", err)
			}

			fmt.Printf("ID: %d\nSource: %d\nForward To: %d\nStart Date: %s\nEnd Date: %s\nUser Keeps Copy: %v\nEnabled: %v\n",
				uf.ID, uf.SourceID, uf.DestID, uf.StartDate, uf.EndDate, uf.UserKeepsCopy, uf.Enabled)

			return nil
		},
	}

	createUserCmd = &cobra.Command{
		Use: "user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if emailAddress == "" {
				return errors.New("no email address provided - pass with flag --email or -e")
			}

			u, err := client.CreateUser(emailAddress)
			if err != nil {
				return err
			}

			fmt.Printf("User created:\nID:%d\nEmail:%s\n", u.ID, u.EmailAddress)
			return nil
		},
	}

	createAPIKeyCmd = &cobra.Command{
		Use:     "api-key",
		Aliases: []string{"key"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if emailAddress == "" {
				return errors.New("no email address provided - pass with flag --email or -e")
			}

			k, err := client.CreateAPIKey(emailAddress)
			if err != nil {
				return err
			}

			fmt.Printf("API key created for %s\nCopy and save this key, it won't be shown again:\n%s\n", emailAddress, k)
			return nil
		},
	}
)

func init() {
	createCmd.AddCommand(createNotifierRuleCmd, createUserCmd, createAPIKeyCmd)
	createNotifierRuleCmd.Flags().IntVarP(&boardID, "board-id", "b", 0, "board id to use")
	createNotifierRuleCmd.Flags().IntVarP(&recipientID, "recipient-id", "r", 0, "recipient id to use")
	createForwardCmd.Flags().BoolVarP(&forwardUserKeeps, "user-keeps-copy", "k", false, "user keeps a copy of forwarded emails")
	createForwardCmd.Flags().IntVarP(&forwardSrcID, "source-id", "s", 0, "source recipient id to forward from")
	createForwardCmd.Flags().IntVarP(&forwardDestID, "dest-id", "d", 0, "destination recipient id to forward to")
	createForwardCmd.Flags().StringVarP(&forwardStartDate, "start-date", "a", "", "start date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().StringVarP(&forwardEndDate, "end-date", "e", "", "end date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().BoolVarP(&forwardEnabled, "enabled", "x", true, "enable the forward")
	createUserCmd.Flags().StringVarP(&emailAddress, "email", "e", "", "email address to create a user for")
	createAPIKeyCmd.Flags().StringVarP(&emailAddress, "email", "e", "", "email address to create an api key for")
}
