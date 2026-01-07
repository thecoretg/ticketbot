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
	rulesModel  *rulesModel
	fwdsModel   *fwdsModel
	entryMode   bool

	help help.Model

	width  int
	height int
}

func NewModel(sl *sdk.Client) *Model {
	rm := newRulesModel(sl)
	fm := newFwdsModel(sl)
	return &Model{
		SDKClient:   sl,
		help:        help.New(),
		activeModel: rm,
		rulesModel:  rm,
		fwdsModel:   fm,
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.rulesModel.Init(), m.fwdsModel.Init())
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
		case !isGlobalKey(msg):
			am, cmd := m.activeModel.Update(msg)
			switch am := am.(type) {
			case *rulesModel:
				m.rulesModel = am
			case *fwdsModel:
				m.fwdsModel = am
			}

			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)

		case key.Matches(msg, allKeys.quit):
			if m.rulesModel.status != rmStatusEntry && m.fwdsModel.status != fwdStatusEntry {
				return m, tea.Quit
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
			if m.activeModel != m.rulesModel {
				m.rulesModel.table.SetCursor(0)
				m.activeModel = m.rulesModel
			}
		case modelTypeFwds:
			if m.activeModel != m.fwdsModel {
				m.fwdsModel.table.SetCursor(0)
				m.activeModel = m.fwdsModel
			}
		}
	}

	rules, cmd := m.rulesModel.Update(msg)
	if r, ok := rules.(*rulesModel); ok {
		m.rulesModel = r
	}
	cmds = append(cmds, cmd)

	fwds, cmd := m.fwdsModel.Update(msg)
	if f, ok := fwds.(*fwdsModel); ok {
		m.fwdsModel = f
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
