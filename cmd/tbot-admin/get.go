package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/models"
)

var (
	getCmd = &cobra.Command{
		Use:               "get",
		PersistentPreRunE: createClient,
	}

	getCfgCmd = &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.GetConfig()
			if err != nil {
				return err
			}

			printCfg(cfg)
			return nil
		},
	}

	getNotifierRuleCmd = &cobra.Command{
		Use:     "notifier-rule",
		Aliases: []string{"rule"},
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := client.GetNotifierRule(id)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nRecipient: %d\nBoard: %d\nNotify: %v\n",
				n.ID, n.WebexRecipientID, n.CwBoardID, n.NotifyEnabled)

			return nil
		},
	}

	getForwardCmd = &cobra.Command{
		Use:     "notifier-forward",
		Aliases: []string{"forward", "fwd"},
		RunE: func(cmd *cobra.Command, args []string) error {
			uf, err := client.GetUserForward(id)
			if err != nil {
				return err
			}

			fmt.Printf("ID: %d\nSource: %d\nForward To: %d\nStart Date: %s\nEnd Date: %s\n"+
				"User Keeps Copy: %v\nEnabled: %v\n",
				uf.ID, uf.SourceID, uf.DestID, uf.StartDate, uf.EndDate, uf.UserKeepsCopy, uf.Enabled)

			return nil
		},
	}
)

func init() {
	getCmd.AddCommand(getCfgCmd, getForwardCmd)
	getNotifierRuleCmd.Flags().IntVar(&id, "id", 0, "id of notifier rule")
	getForwardCmd.Flags().IntVar(&id, "id", 0, "id of forward")
}

func printCfg(cfg *models.Config) {
	fmt.Printf("Attempt Notify: %v\n"+
		"Max Msg Length: %d\n"+
		"Max Concurrent Syncs: %d\n",
		cfg.AttemptNotify, cfg.MaxMessageLength, cfg.MaxConcurrentSyncs)
}
