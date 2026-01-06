package tui

import (
	"sort"

	"github.com/thecoretg/ticketbot/internal/models"
)

func boolToIcon(b bool) string {
	i := "✗"
	if b {
		i = "✓"
	}

	return i
}

func shortenSourceType(s string) string {
	switch s {
	case "person":
		return "p"
	case "room":
		return "r"
	default:
		return "?"
	}
}

func sortBoards(boards []models.Board) {
	sort.SliceStable(boards, func(i, j int) bool {
		return boards[i].Name < boards[j].Name
	})
}

func sortRecips(recips []models.WebexRecipient) {
	sort.SliceStable(recips, func(i, j int) bool {
		return recips[i].Name < recips[j].Name
	})
}
