package cmd

import (
	"errors"
	"os"

	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/service"
)

var (
	serviceCmd = &cobra.Command{
		Use: "service",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if os.Geteuid() != 0 {
				return errors.New("this command must be run as root")
			}
			return nil
		},
	}

	serviceInstallCmd = &cobra.Command{
		Use: "install",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Install()
		},
	}

	serviceStartCmd = &cobra.Command{
		Use: "start",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Start().Run()
		},
	}

	serviceStopCmd = &cobra.Command{
		Use: "stop",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Stop().Run()
		},
	}

	serviceDisableCmd = &cobra.Command{
		Use: "disable",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Disable().Run()
		},
	}

	serviceEnableCmd = &cobra.Command{
		Use: "enable",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Enable().Run()
		},
	}

	serviceRestartCmd = &cobra.Command{
		Use: "restart",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.Restart().Run()
		},
	}

	serviceLogCmd = &cobra.Command{
		Use: "logs",
		RunE: func(cmd *cobra.Command, args []string) error {
			return service.ShowLogs()
		},
	}
)

func addServiceCmd() {
	rootCmd.AddCommand(serviceCmd)
	serviceCmd.AddCommand(serviceInstallCmd, serviceStartCmd, serviceStopCmd, serviceEnableCmd, serviceDisableCmd, serviceRestartCmd, serviceLogCmd)
	serviceCmd.AddCommand(serviceLogCmd)
}
