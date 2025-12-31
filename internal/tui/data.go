package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	sdkErr      struct{ error error }
	gotRulesMsg struct{ rules []models.NotifierRuleFull }
)

func (m *Model) getRules() tea.Cmd {
	return func() tea.Msg {
		rules, err := m.SDKClient.ListNotifierRules()
		if err != nil {
			return sdkErr{error: err}
		}

		return gotRulesMsg{rules: rules}
	}
}
