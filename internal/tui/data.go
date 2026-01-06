package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thecoretg/ticketbot/internal/models"
)

type (
	sdkErr     struct{ error error }
	gotFwdsMsg struct{ fwds []models.NotifierForwardFull }
)

func (m *Model) getFwds() tea.Cmd {
	return func() tea.Msg {
		fwds, err := m.SDKClient.ListUserForwards()
		if err != nil {
			return sdkErr{error: err}
		}

		return gotFwdsMsg{fwds: fwds}
	}
}
