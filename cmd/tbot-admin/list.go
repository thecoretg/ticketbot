package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var (
	listFwdsFilter string
	listCmd        = &cobra.Command{
		Use:               "list",
		PersistentPreRunE: createClient,
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tName")
			for _, b := range boards {
				fmt.Fprintf(w, "%d\t%s\n", b.ID, b.Name)
			}
			w.Flush()
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEnabled\tBoard\tRecipient\tType")
			for _, n := range ns {
				fmt.Fprintf(w, "%d\t%t\t%s\t%s\t%s\n", n.ID, n.Enabled, n.BoardName, n.RecipientName, n.RecipientType)
			}
			w.Flush()
			return nil
		},
	}

	listForwardsCmd = &cobra.Command{
		Use:     "notifier-forwards",
		Aliases: []string{"forwards", "fwds"},
		RunE: func(cmd *cobra.Command, args []string) error {
			params := map[string]string{"filter": strings.ToLower(listFwdsFilter)}
			ufs, err := client.ListUserForwards(params)
			if err != nil {
				return err
			}

			if len(ufs) == 0 {
				fmt.Println("No user forwards found")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEnabled\tStart\tEnd\tKeepCopy\tSource\tSourceType\tDestination\tDestType")
			for _, uf := range ufs {
				sd := "NA"
				ed := "NA"
				if uf.StartDate != nil {
					sd = uf.StartDate.Format("2006-01-02")
				}
				if uf.EndDate != nil {
					ed = uf.EndDate.Format("2006-01-02")
				}

				fmt.Fprintf(w, "%d\t%t\t%s\t%s\t%t\t%s\t%s\t%s\t%s\n",
					uf.ID,
					uf.Enabled,
					sd,
					ed,
					uf.UserKeepsCopy,
					uf.SourceName,
					uf.SourceType,
					uf.DestinationName,
					uf.DestinationType,
				)
			}
			w.Flush()
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tType\tName")
			for _, r := range recips {
				if r.Name != "Empty Title" {
					fmt.Fprintf(w, "%d\t%s\t%s\n", r.ID, r.Type, r.Name)
				}
			}
			w.Flush()
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tEmail\tCreated")
			for _, u := range users {
				fmt.Fprintf(w, "%d\t%s\t%s\n", u.ID, u.EmailAddress, u.CreatedOn.Format("2006-01-02"))
			}
			w.Flush()
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

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "UserID\tKeyID\tCreated")
			for _, k := range keys {
				fmt.Fprintf(w, "%d\t%d\t%s\n", k.UserID, k.ID, k.CreatedOn.Format("2006-01-02"))
			}
			w.Flush()
			return nil
		},
	}
)

func init() {
	listCmd.AddCommand(listBoardsCmd, listNotifierRulesCmd, listForwardsCmd,
		listWebexRecipientsCmd, listUsersCmd, listAPIKeysCmd)
	listForwardsCmd.Flags().StringVarP(&listFwdsFilter, "filter", "f", "", "active filter; valid: 'active', 'inactive', or 'all'")
}

