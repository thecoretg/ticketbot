package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/models"
)

var (
	cfgDebug         bool
	cfgAttemptNotify bool
	cfgMaxMsgLen     int
	cfgMaxSyncs      int

	cfgCmd = &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
	}

	getCfgCmd = &cobra.Command{
		Use: "get",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.GetConfig()
			if err != nil {
				return err
			}

			printCfg(cfg)
			return nil
		},
	}

	updateCfgCmd = &cobra.Command{
		Use: "update",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := client.GetConfig()
			if err != nil {
				return fmt.Errorf("getting current config: %w", err)
			}

			if cmd.Flags().Changed("debug") {
				cfg.Debug = cfgDebug
			}

			if cmd.Flags().Changed("attempt-notify") {
				cfg.AttemptNotify = cfgAttemptNotify
			}

			if cmd.Flags().Changed("max-msg-length") {
				cfg.MaxMessageLength = cfgMaxMsgLen
			}

			if cmd.Flags().Changed("max-concurrent-syncs") {
				cfg.MaxConcurrentSyncs = cfgMaxSyncs
			}

			cfg, err = client.UpdateConfig(cfg)
			if err != nil {
				return err
			}

			fmt.Println("Successfully updated app config. Current config:")
			printCfg(cfg)
			return nil
		},
	}
)

func printCfg(cfg *models.Config) {
	fmt.Printf("Debug: %v\n"+
		"Attempt Notify: %v\n"+
		"Max Msg Length: %d\n"+
		"Max Concurrent Syncs: %d\n",
		cfg.Debug, cfg.AttemptNotify, cfg.MaxMessageLength, cfg.MaxConcurrentSyncs)
}

func init() {
	cfgCmd.AddCommand(getCfgCmd, updateCfgCmd)
	updateCfgCmd.Flags().BoolVarP(&cfgDebug, "debug", "d", false, "enable debug mode on server")
	updateCfgCmd.Flags().BoolVarP(&cfgAttemptNotify, "attempt-notify", "n", false, "attempt notify on server")
	updateCfgCmd.Flags().IntVarP(&cfgMaxMsgLen, "max-msg-length", "l", 300, "max webex message length")
	updateCfgCmd.Flags().IntVarP(&cfgMaxSyncs, "max-concurrent-syncs", "s", 5, "max concurrent syncs")
}
