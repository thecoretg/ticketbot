package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/thecoretg/ticketbot/internal/tui"
)

var adminCmd = &cobra.Command{
	PersistentPreRunE: createClient,
	Use:               "admin",
	RunE: func(cmd *cobra.Command, args []string) error {
		m := tui.NewModel(client)
		if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
			return err
		}

		return nil
	},
}
