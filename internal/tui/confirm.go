package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
)

type (
	confirmDeleteMsg struct{}
)

func completeConfirmForm() tea.Cmd {
	return func() tea.Msg {
		return confirmDeleteMsg{}
	}
}

func confirmationField(prompt string, val *bool) huh.Field {
	opts := []huh.Option[bool]{
		{Key: "No", Value: false},
		{Key: "Yes", Value: true},
	}
	return huh.NewSelect[bool]().
		Title(prompt).
		Options(opts...).
		Value(val)
}

func confirmationForm(prompt string, val *bool, height int) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(confirmationField(prompt, val)),
	).WithTheme(huh.ThemeBase16()).WithHeight(height + 1).WithShowHelp(false)
}
