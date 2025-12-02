package main

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

var (
	notifierID       int
	forwardID        int
	forwardSrcID     int
	forwardDestID    int
	forwardStartDate string
	forwardEndDate   string
	forwardEnabled   bool
	forwardUserKeeps bool

	notifiersCmd = &cobra.Command{
		Use: "notifier",
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

			if len(ns) == 0 {
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
			var (
				p   *models.NotifierRule
				err error
			)

			if boardID == 0 || roomID == 0 {
				p, err = createRuleParamsInteractive(client)
				if err != nil {
					if errors.Is(err, huh.ErrUserAborted) {
						return nil
					}
					return err
				}
			} else {
				p = &models.NotifierRule{
					CwBoardID:   boardID,
					WebexRoomID: roomID,
				}
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

			if len(ufs) == 0 {
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

			fmt.Printf("ID: %d\nSource: %d\nForward To: %d\nStart Date: %s\nEnd Date: %s\n"+
				"User Keeps Copy: %v\nEnabled: %v\n",
				uf.ID, uf.SourceID, uf.DestID, uf.StartDate, uf.EndDate, uf.UserKeepsCopy, uf.Enabled)

			return nil
		},
	}

	createForwardCmd = &cobra.Command{
		Use: "create",
		RunE: func(cmd *cobra.Command, args []string) error {
			if forwardSrcID == 0 {
				return errors.New("source email required")
			}

			if forwardDestID == 0 {
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

func createRuleParamsInteractive(cl *sdk.Client) (*models.NotifierRule, error) {
	boards, err := cl.ListBoards()
	if err != nil {
		return nil, fmt.Errorf("fetching boards: %w", err)
	}

	rooms, err := cl.ListRooms()
	if err != nil {
		return nil, fmt.Errorf("fetching rooms: %w", err)
	}

	// Webex's API returns "Empty Title" for users that have been suspended/deleted. No need to include them.
	rooms = filterEmptyTitleRooms(rooms)

	sort.SliceStable(boards, func(i, j int) bool {
		return boards[i].Name < boards[j].Name
	})

	sort.SliceStable(rooms, func(i, j int) bool {
		return rooms[i].Name < rooms[j].Name
	})

	var bo []huh.Option[models.Board]
	for _, b := range boards {
		opt := huh.Option[models.Board]{
			Key:   b.Name,
			Value: b,
		}
		bo = append(bo, opt)
	}

	var ro []huh.Option[models.WebexRecipient]
	for _, r := range rooms {
		key := fmt.Sprintf("%s (%s)", r.Name, r.Type)
		opt := huh.Option[models.WebexRecipient]{
			Key:   key,
			Value: r,
		}
		ro = append(ro, opt)
	}

	var (
		boardChoice models.Board
		roomChoice  models.WebexRecipient
	)

	bg := huh.NewGroup(
		huh.NewSelect[models.Board]().
			Title("Select a board").
			Options(bo...).
			Value(&boardChoice),
	)

	rg := huh.NewGroup(
		huh.NewSelect[models.WebexRecipient]().
			Title("Select a Webex room").
			Options(ro...).
			Value(&roomChoice),
	)

	f := form(bg, rg)
	if err := f.Run(); err != nil {
		return nil, fmt.Errorf("running form: %w", err)
	}

	n := &models.NotifierRule{
		CwBoardID:     boardChoice.ID,
		WebexRoomID:   roomChoice.ID,
		NotifyEnabled: true,
	}

	return n, nil
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

func init() {
	notifiersCmd.AddCommand(rulesCmd, forwardsCmd)
	rulesCmd.AddCommand(createNotifierRuleCmd, listNotifierRulesCmd, getNotifierRuleCmd, deleteNotifierRuleCmd)
	forwardsCmd.AddCommand(createForwardCmd, listForwardsCmd, getForwardCmd, deleteForwardCmd)
	rulesCmd.PersistentFlags().IntVar(&notifierID, "id", 0, "id of notifier")
	forwardsCmd.PersistentFlags().IntVar(&forwardID, "id", 0, "id of user forward")
	createNotifierRuleCmd.Flags().IntVarP(&boardID, "board-id", "b", 0, "board id to use")
	createNotifierRuleCmd.Flags().IntVarP(&roomID, "room-id", "r", 0, "room id to use")
	createForwardCmd.Flags().BoolVarP(&forwardUserKeeps, "user-keeps-copy", "k", false, "user keeps a copy of forwarded emails")
	createForwardCmd.Flags().IntVarP(&forwardSrcID, "source-id", "s", 0, "source room id to forward from")
	createForwardCmd.Flags().IntVarP(&forwardDestID, "dest-id", "d", 0, "destination room id to forward to")
	createForwardCmd.Flags().StringVarP(&forwardStartDate, "start-date", "a", "", "start date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().StringVarP(&forwardEndDate, "end-date", "e", "", "end date for forward (YYYY-MM-DD)")
	createForwardCmd.Flags().BoolVarP(&forwardEnabled, "enabled", "x", true, "enable the forward")
}
