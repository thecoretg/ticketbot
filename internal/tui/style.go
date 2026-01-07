package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	white    = lipgloss.ANSIColor(7)
	black    = lipgloss.ANSIColor(0)
	grey     = lipgloss.ANSIColor(8)
	green    = lipgloss.ANSIColor(2)
	blue     = lipgloss.ANSIColor(6)
	red      = lipgloss.ANSIColor(1)
	errStyle = lipgloss.NewStyle().Foreground(red).Bold(true)
)

func fillSpaceCentered(content string, w, h int) string {
	return fillSpace(content, w, h, lipgloss.Center, lipgloss.Center)
}

func fillSpace(content string, w, h int, alignH, alignV lipgloss.Position) string {
	return lipgloss.NewStyle().Width(w).Height(h).Align(alignH, alignV).Render(content)
}
