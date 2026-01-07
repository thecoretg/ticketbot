package tui

import "github.com/charmbracelet/bubbles/table"

func newTable() table.Model {
	t := table.New(
		table.WithFocused(true),
	)

	s := table.DefaultStyles()
	s.Header = s.Header.
		Foreground(blue).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(black).
		Background(green).
		Bold(false)

	t.SetStyles(s)
	return t
}
