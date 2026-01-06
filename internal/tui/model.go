package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/thecoretg/ticketbot/pkg/sdk"
)

var currentErr error

type Model struct {
	SDKClient   *sdk.Client
	activeModel tea.Model
	allModels   allModels
	entryMode   bool

	help help.Model

	width  int
	height int
}

type allModels struct {
	rules *rulesModel
	fwds  *fwdsModel
}

func NewModel(sl *sdk.Client) *Model {
	rm := newRulesModel(sl)
	fm := newFwdsModel()
	return &Model{
		SDKClient:   sl,
		help:        help.New(),
		activeModel: rm,
		allModels: allModels{
			rules: rm,
			fwds:  fm,
		},
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.allModels.rules.Init(), m.getFwds())
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		_, hv := m.helpViewSize()
		ev := lipgloss.Height(errView(currentErr))
		cmds = append(cmds, resizeModels(m.width, m.height-hv-ev))
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, allKeys.quit):
			switch m.activeModel {
			case m.allModels.rules:
				if m.allModels.rules.status != rmStatusEntry {
					return m, tea.Quit
				}
			}
		case key.Matches(msg, allKeys.clearErr):
			if currentErr != nil {
				currentErr = nil
			}
		case key.Matches(msg, allKeys.switchModelRules):
			return m, switchModel(modelTypeRules)
		case key.Matches(msg, allKeys.switchModelFwds):
			return m, switchModel(modelTypeFwds)
		}
	case switchModelMsg:
		switch msg.modelType {
		case modelTypeRules:
			if m.activeModel != m.allModels.rules {
				m.allModels.rules.table.SetCursor(0)
				m.activeModel = m.allModels.rules
			}
		case modelTypeFwds:
			if m.activeModel != m.allModels.fwds {
				m.allModels.fwds.table.SetCursor(0)
				m.activeModel = m.allModels.fwds
			}
		}
	}

	rules, cmd := m.allModels.rules.Update(msg)
	if r, ok := rules.(*rulesModel); ok {
		m.allModels.rules = r
	}
	cmds = append(cmds, cmd)

	fwds, cmd := m.allModels.fwds.Update(msg)
	if f, ok := fwds.(*fwdsModel); ok {
		m.allModels.fwds = f
	}
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m *Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Top, m.activeModel.View(), errView(currentErr), m.helpView())
}

func errView(err error) string {
	if err != nil {
		return errStyle.Render("Error: " + err.Error())
	}
	return ""
}
