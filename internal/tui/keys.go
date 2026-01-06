package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type keyMap struct {
	quit             key.Binding
	switchModelRules key.Binding
	switchModelFwds  key.Binding
	newItem          key.Binding
	deleteItem       key.Binding
}

var allKeys = keyMap{
	quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
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
		case m.allModels.rules:
			if len(m.allModels.rules.rules) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelFwds)
		case m.allModels.fwds:
			if len(m.allModels.fwds.fwds) > 0 {
				keys = append(keys, allKeys.deleteItem)
			}
			keys = append(keys, allKeys.switchModelRules)
		}
	}

	return keys
}

func (m *Model) helpViewSize() (int, int) {
	hv := m.helpView()
	return lipgloss.Width(hv), lipgloss.Height(hv)
}

func (m *Model) helpView() string {
	if m.activeModel == m.allModels.rules && m.allModels.rules.status == rmStatusEntry {
		f := m.allModels.rules.form
		return f.Help().ShortHelpView(f.KeyBinds())
	}
	return m.help.ShortHelpView(m.helpKeys())
}
