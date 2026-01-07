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
		key.WithHelp("ctrl+r", "rules"),
	),
	switchModelFwds: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "forwards"),
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

	if !m.entryMode {
		switch m.activeModel {
		case m.rulesModel:
			if len(m.rulesModel.rules) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelFwds)
		case m.fwdsModel:
			if len(m.fwdsModel.fwds) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelRules)
		}
	}

	if currentErr != nil {
		keys = append(keys, allKeys.clearErr)
	}

	return keys
}

func (m *Model) helpViewSize() (int, int) {
	hv := m.helpView()
	return lipgloss.Width(hv), lipgloss.Height(hv)
}

func (m *Model) helpView() string {
	if m.activeModel == m.rulesModel && m.rulesModel.status == rmStatusEntry {
		// add quit bind to form help
		f := m.rulesModel.form
		k := append(f.KeyBinds(), allKeys.quit)
		return f.Help().ShortHelpView(k)
	}
	return m.help.ShortHelpView(m.helpKeys())
}

func isGlobalKey(msg tea.KeyMsg) bool {
	return key.Matches(msg, allKeys.quit) ||
		key.Matches(msg, allKeys.clearErr) ||
		key.Matches(msg, allKeys.switchModelRules) ||
		key.Matches(msg, allKeys.switchModelFwds)
}
