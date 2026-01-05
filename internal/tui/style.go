package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	white     = lipgloss.ANSIColor(7)
	grey      = lipgloss.ANSIColor(8)
	blue      = lipgloss.ANSIColor(4)
	red       = lipgloss.ANSIColor(1)
	helpStyle = lipgloss.NewStyle().Foreground(white)
	errStyle  = lipgloss.NewStyle().Foreground(red).Bold(true)
	divStyle  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false)
)

func dividerHeight() int {
	_, fh := divStyle.GetFrameSize()
	h := lipgloss.Height(divider(1))
	return fh + h
}

func divider(w int) string {
	return lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, false, false, false).Width(w).Render()
}
