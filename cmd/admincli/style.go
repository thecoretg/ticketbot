package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var (
	spacingStyle = lipgloss.NewStyle().Padding(0, 1)
)

func spacingStyleFunc() table.StyleFunc {
	return func(row, col int) lipgloss.Style {
		return spacingStyle
	}
}
