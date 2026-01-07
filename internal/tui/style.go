package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

var (
	white                = lipgloss.ANSIColor(7)
	black                = lipgloss.ANSIColor(0)
	green                = lipgloss.ANSIColor(2)
	blue                 = lipgloss.ANSIColor(6)
	red                  = lipgloss.ANSIColor(1)
	errStyle             = lipgloss.NewStyle().Foreground(red).Bold(true)
	menuLabelStyle       = lipgloss.NewStyle().Foreground(white)
	activeMenuLabelStyle = lipgloss.NewStyle().Foreground(green).Bold(true)
)

func fillSpaceCentered(content string, w, h int) string {
	return fillSpace(content, w, h, lipgloss.Center, lipgloss.Center)
}

func fillSpace(content string, w, h int, alignH, alignV lipgloss.Position) string {
	return lipgloss.NewStyle().Width(w).Height(h).Align(alignH, alignV).Render(content)
}

func useSpinner(s spinner.Model, text string) string {
	return fmt.Sprintf("%s %s", s.View(), text)
}

func newSpinner(s spinner.Spinner, color lipgloss.ANSIColor) spinner.Model {
	style := lipgloss.NewStyle().Foreground(color)
	return spinner.New(spinner.WithSpinner(s), spinner.WithStyle(style))
}
