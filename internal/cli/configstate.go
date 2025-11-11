package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/server"
)

var (
	cfgDebug           bool
	attemptNotify      bool
	maxMsgLength       int
	maxConcurrentSyncs int

	stateCmd = &cobra.Command{
		Use: "state",
		RunE: func(cmd *cobra.Command, args []string) error {
			state, err := client.GetAppState()
			if err != nil {
				return fmt.Errorf("getting app state: %w", err)
			}

			fmt.Printf("Syncing Tickets: %v\n"+
				"Syncing Rooms: %v\n"+
				"Syncing Boards: %v\n",
				state.SyncingTickets, state.SyncingWebexRooms, state.SyncingBoards)

			return nil
		},
	}

	cfgCmd = &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
	}

	getCfgCmd = &cobra.Command{
		Use: "get",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.GetConfig()
			if err != nil {
				return fmt.Errorf("getting current config: %w", err)
			}

			printCfg(cfg)
			return nil
		},
	}

	updateCfgCmd = &cobra.Command{
		Use: "update",
		RunE: func(cmd *cobra.Command, args []string) error {
			p := &server.AppConfigPayload{
				Debug:              flagToBoolPtr(cmd, "debug", cfgDebug),
				AttemptNotify:      flagToBoolPtr(cmd, "attempt-notify", attemptNotify),
				MaxMessageLength:   flagToIntPtr(cmd, "max-msg-length", maxMsgLength),
				MaxConcurrentSyncs: flagToIntPtr(cmd, "max-concurrent-syncs", maxConcurrentSyncs),
			}

			cfg, err := client.UpdateConfig(p)
			if err != nil {
				return fmt.Errorf("updating config: %w", err)
			}

			fmt.Println("Successfully updated app config. Current config:")
			printCfg(cfg)
			return nil
		},
	}
)

func printCfg(cfg *server.AppConfig) {
	fmt.Printf("Debug: %v\n"+
		"Attempt Notify: %v\n"+
		"Max Msg Length: %d\n"+
		"Max Concurrent Syncs: %d\n",
		cfg.Debug, cfg.AttemptNotify, cfg.MaxMessageLength, cfg.MaxConcurrentSyncs)
}

func init() {
	cfgCmd.AddCommand(getCfgCmd, updateCfgCmd)
	updateCfgCmd.Flags().BoolVarP(&cfgDebug, "debug", "d", false, "enable debug mode on server")
	updateCfgCmd.Flags().BoolVarP(&attemptNotify, "attempt-notify", "n", false, "attempt notify on server")
	updateCfgCmd.Flags().IntVarP(&maxMsgLength, "max-msg-length", "l", 300, "max webex message length")
	updateCfgCmd.Flags().IntVarP(&maxConcurrentSyncs, "max-concurrent-syncs", "s", 5, "max concurrent syncs")
}
