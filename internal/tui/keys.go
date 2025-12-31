package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	quit key.Binding
	up   key.Binding
	down key.Binding
}

var defaultKeyMap = keyMap{
	quit: key.NewBinding(
		key.WithKeys("q"),
		key.WithHelp("q", "quit"),
	),
	up: key.NewBinding(
		key.WithKeys("k", "up"),
	),
	down: key.NewBinding(
		key.WithKeys("j", "down"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.quit},
	}
}
