package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	quit             key.Binding
	clearErr         key.Binding
	switchModelRules key.Binding
	switchModelFwds  key.Binding
	switchModelUsers key.Binding
	newItem          key.Binding
	deleteItem       key.Binding
}

var allKeys = keyMap{
	quit: key.NewBinding(
		key.WithKeys("ctrl+c"),
		key.WithHelp("ctrl+c", "quit"),
	),
	clearErr: key.NewBinding(
		key.WithKeys("ctrl+e"),
		key.WithHelp("ctrl+e", "clear error"),
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
	keys = append(keys, allKeys.quit, allKeys.newItem)

	if currentErr != nil {
		keys = append(keys, allKeys.clearErr)
	}

	if m.activeModel == nil {
		return keys
	}

	if m.activeModel.Status() == statusMain && len(m.activeModel.Table().Rows()) != 0 {
		switch m.activeModel {
		case m.rulesModel:
			if len(m.rulesModel.rules) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelFwds, allKeys.switchModelUsers)
		case m.fwdsModel:
			if len(m.fwdsModel.fwds) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelRules, allKeys.switchModelUsers)
		case m.usersModel:
			if len(m.usersModel.users) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelRules, allKeys.switchModelFwds)
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
		key.Matches(msg, allKeys.clearErr) ||
		key.Matches(msg, allKeys.switchModelRules) ||
		key.Matches(msg, allKeys.switchModelFwds) ||
		key.Matches(msg, allKeys.switchModelUsers)
}
