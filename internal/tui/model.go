package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/thecoretg/ticketbot/internal/models"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

type Model struct {
	SDKClient   *sdk.Client
	activeModel tea.Model
	allModels   allModels
	data        *data
	keys        keyMap
}

type allModels struct {
	rules *rulesModel
}

type data struct {
	rules []models.NotifierRuleFull
}

func NewModel(sl *sdk.Client) *Model {
	rm := newRulesModel()
	return &Model{
		SDKClient:   sl,
		keys:        defaultKeyMap,
		activeModel: rm,
		allModels: allModels{
			rules: rm,
		},
		data: &data{
			rules: []models.NotifierRuleFull{},
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return m.getRules()
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.quit):
			return m, tea.Quit
		}
	}

	rules, cmd := m.allModels.rules.Update(msg)
	if r, ok := rules.(*rulesModel); ok {
		m.allModels.rules = r
	}

	return m, cmd
}

func (m *Model) View() string {
	return m.activeModel.View()
}
