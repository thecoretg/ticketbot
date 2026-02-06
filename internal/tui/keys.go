package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	quit               key.Binding
	nextModel          key.Binding
	prevModel          key.Binding
	switchModelRules   key.Binding
	switchModelFwds    key.Binding
	switchModelUsers   key.Binding
	switchModelAPIKeys key.Binding
	switchModelSync    key.Binding
	newItem            key.Binding
	deleteItem         key.Binding
}

var allKeys = keyMap{
	quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	nextModel: key.NewBinding(
		key.WithKeys("ctrl+l", "right"),
		key.WithHelp("right", "next tab"),
	),
	prevModel: key.NewBinding(
		key.WithKeys("ctrl+h", "left"),
		key.WithHelp("left", "prev tab"),
	),
	switchModelRules: key.NewBinding(
		key.WithKeys("ctrl+r"),
	),
	switchModelFwds: key.NewBinding(
		key.WithKeys("ctrl+f"),
	),
	switchModelUsers: key.NewBinding(
		key.WithKeys("ctrl+u"),
	),
	switchModelAPIKeys: key.NewBinding(
		key.WithKeys("ctrl+a"),
	),
	switchModelSync: key.NewBinding(
		key.WithKeys("ctrl+s"),
	),
	newItem: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	deleteItem: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "delete"),
	),
}

// ShortHelp() is here to satisfy an interface
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{}
}

// FullHelp() is here to satisfy an interface
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{}
}

func (m *Model) helpKeys() []key.Binding {
	var keys []key.Binding
	keys = append(keys, allKeys.quit, allKeys.newItem, allKeys.prevModel, allKeys.nextModel)

	if m.activeModel == nil {
		return keys
	}

	if m.activeModel.Status() == statusMain && len(m.activeModel.Table().Rows()) != 0 {
		switch m.activeModel {
		case m.rulesModel:
			if len(m.rulesModel.rules) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
		case m.fwdsModel:
			if len(m.fwdsModel.fwds) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
		case m.usersModel:
			if len(m.usersModel.users) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
		case m.apiKeysModel:
			if len(m.apiKeysModel.keys) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
		}
	}

	return keys
}

func (m *Model) helpViewSize() (int, int) {
	hv := m.helpView()
	return lipgloss.Width(hv), lipgloss.Height(hv)
}

func (m *Model) helpView() string {
	if m.activeModel != nil && m.activeModel.Status().inForm() {
		// add quit bind to form help
		f := m.activeModel.Form()
		k := append(f.KeyBinds(), allKeys.quit)
		return f.Help().ShortHelpView(k)
	}
	return m.help.ShortHelpView(m.helpKeys())
}

func isGlobalKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, allKeys.quit) ||
		key.Matches(msg, allKeys.nextModel) ||
		key.Matches(msg, allKeys.prevModel) ||
		key.Matches(msg, allKeys.switchModelRules) ||
		key.Matches(msg, allKeys.switchModelFwds) ||
		key.Matches(msg, allKeys.switchModelUsers) ||
		key.Matches(msg, allKeys.switchModelAPIKeys) ||
		key.Matches(msg, allKeys.switchModelSync)
}
