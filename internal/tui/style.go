package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	white           = lipgloss.ANSIColor(7)
	grey            = lipgloss.ANSIColor(8)
	border          = lipgloss.NormalBorder()
	borderStyle     = lipgloss.NewStyle().Foreground(grey)
	headerStyle     = lipgloss.NewStyle().Foreground(white).Align(lipgloss.Center)
	cellStyle       = lipgloss.NewStyle().Padding(0, 1).Align(lipgloss.Center)
	selectedStyle   = cellStyle.Foreground(white)
	unselectedStyle = cellStyle.Foreground(grey)
)
